package evmd

import (
	ibcratelimiterext "github.com/cosmos/evm/x/ibcratelimiterext"
	ratelimiting "github.com/cosmos/ibc-apps/modules/rate-limiting/v10"
	ratelimitkeeper "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/keeper"
	ratelimittypes "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/types"
	ratelimitingv2 "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/v2"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcapi "github.com/cosmos/ibc-go/v10/modules/core/api"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/module"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

func optionalRateLimitStoreKeys() []string {
	return []string{ratelimittypes.StoreKey}
}

func setupOptionalRateLimit(
	app *EVMD,
	appCodec codec.Codec,
	keys map[string]*storetypes.KVStoreKey,
	authAddr string,
	transferStack porttypes.IBCModule,
	transferStackV2 ibcapi.IBCModule,
) (porttypes.IBCModule, ibcapi.IBCModule, module.AppModule) {
	rateLimitKeeper := ratelimitkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[ratelimittypes.StoreKey]),
		paramtypes.Subspace{},
		// NOTE: ratelimit params are currently empty, so an empty subspace is sufficient.
		authAddr,
		app.BankKeeper,
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.ClientKeeper,
		app.IBCKeeper.ChannelKeeper,
	)

	rateLimitMiddleware := ratelimiting.NewIBCMiddleware(*rateLimitKeeper, transferStack)
	transferStack = rateLimitMiddleware
	transferStackV2 = ratelimitingv2.NewIBCMiddleware(*rateLimitKeeper, transferStackV2)

	return transferStack, transferStackV2, ibcratelimiterext.NewRateLimitingAppModule(rateLimitKeeper, app.IbcRateLimiterExtKeeper)
}

func optionalRateLimitBeginBlockers() []string {
	return []string{ratelimittypes.ModuleName}
}

func optionalRateLimitGenesisModules() []string {
	return []string{ratelimittypes.ModuleName}
}
