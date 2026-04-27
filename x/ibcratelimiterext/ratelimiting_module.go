package ibcratelimiterext

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	"cosmossdk.io/core/appmodule"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	extkeeper "github.com/cosmos/evm/x/ibcratelimiterext/keeper"
	ibcratelimiterextkeeper "github.com/cosmos/evm/x/ibcratelimiterext/keeper"
	ratelimitcli "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/client/cli"
	ratelimitkeeper "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/keeper"
	ratelimittypes "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/types"
)

var (
	_ module.AppModule           = (*RateLimitingAppModule)(nil)
	_ module.AppModuleBasic      = (*RateLimitingAppModuleBasic)(nil)
	_ module.HasGenesis          = (*RateLimitingAppModule)(nil)
	_ module.HasName             = (*RateLimitingAppModule)(nil)
	_ module.HasConsensusVersion = (*RateLimitingAppModule)(nil)
	_ module.HasServices         = (*RateLimitingAppModule)(nil)
	_ appmodule.AppModule        = (*RateLimitingAppModule)(nil)
	_ appmodule.HasBeginBlocker  = (*RateLimitingAppModule)(nil)
)

type RateLimitingAppModuleBasic struct{}

func (RateLimitingAppModuleBasic) Name() string   { return ratelimittypes.ModuleName }
func (RateLimitingAppModule) IsOnePerModuleType() {}
func (RateLimitingAppModule) IsAppModule()        {}

func (RateLimitingAppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	ratelimittypes.RegisterLegacyAminoCodec(cdc)
}

func (RateLimitingAppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	ratelimittypes.RegisterInterfaces(registry)
}

func (RateLimitingAppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(ratelimittypes.DefaultGenesis())
}

func (RateLimitingAppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var gs ratelimittypes.GenesisState
	if err := cdc.UnmarshalJSON(bz, &gs); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", ratelimittypes.ModuleName, err)
	}
	return gs.Validate()
}

func (RateLimitingAppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	if err := ratelimittypes.RegisterQueryHandlerClient(context.Background(), mux, ratelimittypes.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

func (RateLimitingAppModuleBasic) GetTxCmd() *cobra.Command { return nil }
func (RateLimitingAppModuleBasic) GetQueryCmd() *cobra.Command {
	return ratelimitcli.GetQueryCmd()
}

type RateLimitingAppModule struct {
	RateLimitingAppModuleBasic
	rateLimitKeeper *ratelimitkeeper.Keeper
	msgServer       ratelimittypes.MsgServer
}

func NewRateLimitingAppModule(rateLimitKeeper *ratelimitkeeper.Keeper, extKeeper extkeeper.Keeper) RateLimitingAppModule {
	return RateLimitingAppModule{
		RateLimitingAppModuleBasic: RateLimitingAppModuleBasic{},
		rateLimitKeeper:            rateLimitKeeper,
		msgServer:                  ibcratelimiterextkeeper.NewRateLimitMsgServer(rateLimitKeeper, extKeeper),
	}
}

func (am RateLimitingAppModule) RegisterServices(cfg module.Configurator) {
	ratelimittypes.RegisterMsgServer(cfg.MsgServer(), am.msgServer)
	ratelimittypes.RegisterQueryServer(cfg.QueryServer(), am.rateLimitKeeper)
}

func (am RateLimitingAppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) {
	var genesisState ratelimittypes.GenesisState
	cdc.MustUnmarshalJSON(data, &genesisState)
	am.rateLimitKeeper.InitGenesis(ctx, genesisState)
}

func (am RateLimitingAppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs := am.rateLimitKeeper.ExportGenesis(ctx)
	return cdc.MustMarshalJSON(gs)
}

func (RateLimitingAppModule) ConsensusVersion() uint64 { return 1 }

func (am RateLimitingAppModule) BeginBlock(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	am.rateLimitKeeper.BeginBlocker(sdkCtx)
	return nil
}
