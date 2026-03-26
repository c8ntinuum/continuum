package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	cmttypes "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/evm/x/ibcratelimiterext/types"

	"cosmossdk.io/log"
	store "cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func setupKeeper(t *testing.T) (Keeper, sdk.Context) {
	t.Helper()
	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount("cosmos", "cosmospub")

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmttypes.Header{}, false, log.NewNopLogger())
	ir := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(ir)
	authority := sdk.AccAddress([]byte("authority_addr_12345"))

	k := NewKeeper(cdc, storeKey, authority)
	return k, ctx
}

func TestParamsRoundTrip(t *testing.T) {
	k, ctx := setupKeeper(t)
	params := types.Params{Whitelist: []string{sdk.AccAddress([]byte("operator_1")).String()}}

	require.NoError(t, params.Validate())
	k.SetParams(ctx, params)
	got := k.GetParams(ctx)
	require.Equal(t, params, got)
}

func TestIsWhitelisted(t *testing.T) {
	k, ctx := setupKeeper(t)
	addr := sdk.AccAddress([]byte("operator_1"))
	k.SetParams(ctx, types.Params{Whitelist: []string{addr.String()}})
	require.True(t, k.IsWhitelisted(ctx, addr))
	require.False(t, k.IsWhitelisted(ctx, sdk.AccAddress([]byte("operator_2"))))
}
