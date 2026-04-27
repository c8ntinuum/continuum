package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	cmttypes "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/evm/x/ibcbreaker/types"

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

	return NewKeeper(cdc, storeKey, authority), ctx
}

func TestUpdateIbcBreakerSkipsNoOpStateWrite(t *testing.T) {
	k, ctx := setupKeeper(t)
	srv := NewMsgServerImpl(k)

	operator := sdk.AccAddress([]byte("operator_1"))
	k.SetParams(ctx, types.Params{Whitelist: []string{operator.String()}})
	k.SetIbcAvailable(ctx, true)

	_, err := srv.UpdateIbcBreaker(sdk.WrapSDKContext(ctx), &types.MsgUpdateIbcBreaker{
		Signer:       operator.String(),
		IbcAvailable: true,
	})
	require.NoError(t, err)
	require.True(t, k.GetIbcAvailable(ctx))

	_, err = srv.UpdateIbcBreaker(sdk.WrapSDKContext(ctx), &types.MsgUpdateIbcBreaker{
		Signer:       operator.String(),
		IbcAvailable: false,
	})
	require.NoError(t, err)
	require.False(t, k.GetIbcAvailable(ctx))
}
