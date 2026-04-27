package keeper

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	cmttypes "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/evm/x/circuit/types"

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

func TestUpdateParamsAuthorityCanonicalization(t *testing.T) {
	k, ctx := setupKeeper(t)
	srv := NewMsgServerImpl(k)

	whitelisted := sdk.AccAddress([]byte("operator_1")).String()

	_, err := srv.UpdateParams(sdk.WrapSDKContext(ctx), &types.MsgUpdateParams{
		Authority: strings.ToUpper(k.authority.String()),
		Params:    &types.Params{Whitelist: []string{whitelisted}},
	})
	require.NoError(t, err)
	require.Equal(t, []string{whitelisted}, k.GetParams(ctx).Whitelist)

	_, err = srv.UpdateParams(sdk.WrapSDKContext(ctx), &types.MsgUpdateParams{
		Authority: sdk.AccAddress([]byte("bad_authority")).String(),
		Params:    &types.Params{Whitelist: []string{whitelisted}},
	})
	require.Error(t, err)
}

func TestUpdateCircuitSkipsNoOpStateWrite(t *testing.T) {
	k, ctx := setupKeeper(t)
	srv := NewMsgServerImpl(k)

	operator := sdk.AccAddress([]byte("operator_1"))
	k.SetParams(ctx, types.Params{Whitelist: []string{operator.String()}})
	k.SetSystemAvailable(ctx, true)

	_, err := srv.UpdateCircuit(sdk.WrapSDKContext(ctx), &types.MsgUpdateCircuit{
		Signer:          operator.String(),
		SystemAvailable: true,
	})
	require.NoError(t, err)
	require.True(t, k.GetSystemAvailable(ctx))

	_, err = srv.UpdateCircuit(sdk.WrapSDKContext(ctx), &types.MsgUpdateCircuit{
		Signer:          operator.String(),
		SystemAvailable: false,
	})
	require.NoError(t, err)
	require.False(t, k.GetSystemAvailable(ctx))
}

func TestUpdateParamsRejectsEmptyWhitelistWhileSystemUnavailable(t *testing.T) {
	k, ctx := setupKeeper(t)
	srv := NewMsgServerImpl(k)

	k.SetSystemAvailable(ctx, false)

	_, err := srv.UpdateParams(sdk.WrapSDKContext(ctx), &types.MsgUpdateParams{
		Authority: k.authority.String(),
		Params:    &types.Params{Whitelist: []string{}},
	})
	require.Error(t, err)

	operator := sdk.AccAddress([]byte("operator_1")).String()
	_, err = srv.UpdateParams(sdk.WrapSDKContext(ctx), &types.MsgUpdateParams{
		Authority: k.authority.String(),
		Params:    &types.Params{Whitelist: []string{operator}},
	})
	require.NoError(t, err)

	k.SetSystemAvailable(ctx, true)
	_, err = srv.UpdateParams(sdk.WrapSDKContext(ctx), &types.MsgUpdateParams{
		Authority: k.authority.String(),
		Params:    &types.Params{Whitelist: []string{}},
	})
	require.NoError(t, err)
	require.Empty(t, k.GetParams(ctx).Whitelist)
}

func TestUpdateCircuitRejectsNonWhitelistedSigner(t *testing.T) {
	k, ctx := setupKeeper(t)
	srv := NewMsgServerImpl(k)

	allowed := sdk.AccAddress([]byte("operator_1"))
	blocked := sdk.AccAddress([]byte("operator_2"))
	k.SetParams(ctx, types.Params{Whitelist: []string{allowed.String()}})

	_, err := srv.UpdateCircuit(sdk.WrapSDKContext(ctx), &types.MsgUpdateCircuit{
		Signer:          blocked.String(),
		SystemAvailable: false,
	})
	require.Error(t, err)
	require.True(t, k.GetSystemAvailable(ctx))
}

func TestUpdateCircuitRejectsNilRequest(t *testing.T) {
	k, ctx := setupKeeper(t)
	srv := NewMsgServerImpl(k)

	_, err := srv.UpdateCircuit(sdk.WrapSDKContext(ctx), nil)
	require.Error(t, err)
}

func TestUpdateParamsRejectsNilRequest(t *testing.T) {
	k, ctx := setupKeeper(t)
	srv := NewMsgServerImpl(k)

	_, err := srv.UpdateParams(sdk.WrapSDKContext(ctx), nil)
	require.Error(t, err)
}

func TestSystemAvailableRejectsNilRequest(t *testing.T) {
	k, ctx := setupKeeper(t)

	_, err := k.SystemAvailable(sdk.WrapSDKContext(ctx), nil)
	require.Error(t, err)
}

func TestWhitelistRejectsNilRequest(t *testing.T) {
	k, ctx := setupKeeper(t)

	_, err := k.Whitelist(sdk.WrapSDKContext(ctx), nil)
	require.Error(t, err)
}
