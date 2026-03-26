package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/evm/x/ibcratelimiterext/types"
)

func TestUpdateParams_AuthorityOnly(t *testing.T) {
	k, ctx := setupKeeper(t)
	srv := NewMsgServerImpl(k)

	good := sdk.AccAddress([]byte("good_operator")).String()

	_, err := srv.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: k.authority.String(),
		Params:    &types.Params{Whitelist: []string{good}},
	})
	require.NoError(t, err)
	require.Equal(t, []string{good}, k.GetParams(ctx).Whitelist)

	_, err = srv.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: sdk.AccAddress([]byte("bad_authority")).String(),
		Params:    &types.Params{Whitelist: []string{good}},
	})
	require.Error(t, err)
}
