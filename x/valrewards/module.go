package valrewards

import (
	"context"
	"encoding/json"
	"fmt"

	"cosmossdk.io/core/appmodule"
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

func (AppModuleBasic) RegisterLegacyAminoCodec(_ *codec.LegacyAmino) {}

func (AppModuleBasic) ConsensusVersion() uint64 {
	return consensusVersion
}

func (AppModuleBasic) RegisterInterfaces(_ codectypes.InterfaceRegistry) {}

func (AppModuleBasic) DefaultGenesis(_ codec.JSONCodec) json.RawMessage {
	return nil
}

func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, _ json.RawMessage) error {
	return nil
}

func (AppModuleBasic) RegisterRESTRoutes(_ client.Context, _ *mux.Router) {}

func (b AppModuleBasic) RegisterGRPCGatewayRoutes(_ client.Context, _ *runtime.ServeMux) {}

func (AppModuleBasic) GetTxCmd() *cobra.Command { return nil }

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

func (AppModule) RegisterServices(_ module.Configurator) {}

func (am AppModule) InitGenesis(ctx sdk.Context, _ codec.JSONCodec, _ json.RawMessage) []abci.ValidatorUpdate {
	if moduleAcc := am.accountKeeper.GetModuleAccount(ctx, vrtypes.ModuleName); moduleAcc == nil {
		panic(fmt.Sprintf("%s module account has not been set", vrtypes.ModuleName))
	}
	return []abci.ValidatorUpdate{}
}

func (AppModule) ExportGenesis(_ sdk.Context, _ codec.JSONCodec) json.RawMessage {
	return nil
}

func (AppModule) GenerateGenesisState(_ *module.SimulationState) {}

func (am AppModule) BeginBlock(ctx context.Context) error {
	return am.keeper.BeginBlocker(sdk.UnwrapSDKContext(ctx))
}

func (AppModule) EndBlock(ctx context.Context) error { return nil }

func (AppModule) IsAppModule() {}

func (AppModule) IsOnePerModuleType() {}
