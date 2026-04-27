package valrewards

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	vrcli "github.com/cosmos/evm/x/valrewards/client/cli"
	vrkeeper "github.com/cosmos/evm/x/valrewards/keeper"
	vrtypes "github.com/cosmos/evm/x/valrewards/types"
)

var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
	_ module.HasABCIGenesis = AppModule{}

	_ appmodule.HasBeginBlocker = AppModule{}
	_ appmodule.HasEndBlocker   = AppModule{}
	_ appmodule.AppModule       = AppModule{}
)

const consensusVersion = 1

type AppModuleBasic struct{}

func (AppModuleBasic) Name() string { return vrtypes.ModuleName }

func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	vrtypes.RegisterLegacyAminoCodec(cdc)
}

func (AppModuleBasic) ConsensusVersion() uint64 {
	return consensusVersion
}

func (AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	vrtypes.RegisterInterfaces(registry)
}

func (AppModuleBasic) DefaultGenesis(_ codec.JSONCodec) json.RawMessage {
	return mustMarshalGenesisState(vrtypes.DefaultGenesisState())
}

func (AppModuleBasic) ValidateGenesis(_ codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	gs, err := parseGenesisState(bz)
	if err != nil {
		return err
	}
	return gs.Validate()
}

func (AppModuleBasic) RegisterRESTRoutes(_ client.Context, _ *mux.Router) {}

func (b AppModuleBasic) RegisterGRPCGatewayRoutes(_ client.Context, _ *runtime.ServeMux) {}

func (AppModuleBasic) GetTxCmd() *cobra.Command { return vrcli.NewTxCmd() }

func (AppModuleBasic) GetQueryCmd() *cobra.Command { return nil }

type AppModule struct {
	AppModuleBasic
	keeper        vrkeeper.Keeper
	accountKeeper authkeeper.AccountKeeper
}

func NewAppModule(k vrkeeper.Keeper, ak authkeeper.AccountKeeper) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{},
		keeper:         k,
		accountKeeper:  ak,
	}
}

func (AppModule) Name() string {
	return vrtypes.ModuleName
}

func (am AppModule) RegisterServices(cfg module.Configurator) {
	vrtypes.RegisterMsgServer(cfg.MsgServer(), am.keeper)
	vrtypes.RegisterQueryServer(cfg.QueryServer(), am.keeper)
}

func (am AppModule) InitGenesis(ctx sdk.Context, _ codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	if moduleAcc := am.accountKeeper.GetModuleAccount(ctx, vrtypes.ModuleName); moduleAcc == nil {
		panic(fmt.Sprintf("%s module account has not been set", vrtypes.ModuleName))
	}

	gs, err := parseGenesisState(data)
	if err != nil {
		panic(fmt.Sprintf("failed to parse %s genesis state: %v", vrtypes.ModuleName, err))
	}
	if err := gs.Validate(); err != nil {
		panic(fmt.Sprintf("invalid %s genesis state: %v", vrtypes.ModuleName, err))
	}
	if err := validateOutstandingRewardsFunding(gs, am.keeper.GetRewardsPool(ctx)); err != nil {
		panic(fmt.Sprintf("invalid %s genesis state: %v", vrtypes.ModuleName, err))
	}

	InitGenesis(ctx, am.keeper, gs)
	return []abci.ValidatorUpdate{}
}

func (am AppModule) ExportGenesis(ctx sdk.Context, _ codec.JSONCodec) json.RawMessage {
	gs := ExportGenesis(ctx, am.keeper)
	return mustMarshalGenesisState(gs)
}

func (AppModule) GenerateGenesisState(_ *module.SimulationState) {}

func (am AppModule) BeginBlock(ctx context.Context) error {
	return am.keeper.BeginBlocker(sdk.UnwrapSDKContext(ctx))
}

func (AppModule) EndBlock(ctx context.Context) error { return nil }

func (AppModule) IsAppModule() {}

func (AppModule) IsOnePerModuleType() {}

func parseGenesisState(bz json.RawMessage) (vrtypes.GenesisState, error) {
	trimmed := bytes.TrimSpace(bz)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return *vrtypes.DefaultGenesisState(), nil
	}

	gs := *vrtypes.DefaultGenesisState()
	if err := json.Unmarshal(trimmed, &gs); err != nil {
		return vrtypes.GenesisState{}, fmt.Errorf("failed to unmarshal %s genesis state: %w", vrtypes.ModuleName, err)
	}
	if gs.ValidatorPoints == nil {
		gs.ValidatorPoints = []vrtypes.GenesisValidatorPoint{}
	}
	if gs.ValidatorOutstandingRewards == nil {
		gs.ValidatorOutstandingRewards = []vrtypes.GenesisValidatorOutstandingReward{}
	}
	if gs.Params.Whitelist == nil {
		gs.Params.Whitelist = []string{}
	}

	return gs, nil
}

func mustMarshalGenesisState(gs *vrtypes.GenesisState) json.RawMessage {
	bz, err := json.Marshal(gs)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal %s genesis state: %v", vrtypes.ModuleName, err))
	}
	return bz
}

func validateOutstandingRewardsFunding(gs vrtypes.GenesisState, pool sdk.Coin) error {
	totalOutstanding := math.ZeroInt()
	for _, entry := range gs.ValidatorOutstandingRewards {
		if entry.Amount.Denom != pool.Denom {
			return fmt.Errorf(
				"invalid outstanding reward denom for epoch %d and validator %s: got %s, expected %s",
				entry.Epoch,
				entry.ValidatorAddress,
				entry.Amount.Denom,
				pool.Denom,
			)
		}
		totalOutstanding = totalOutstanding.Add(entry.Amount.Amount)
	}

	required := sdk.NewCoin(pool.Denom, totalOutstanding)
	if pool.IsLT(required) {
		return fmt.Errorf(
			"outstanding rewards exceed funded pool balance: outstanding=%s pool=%s",
			required.String(),
			pool.String(),
		)
	}

	return nil
}
