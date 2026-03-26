package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/evm/x/ibcratelimiterext/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) Whitelist(c context.Context, _ *types.QueryWhitelistRequest) (*types.QueryWhitelistResponse, error) {
	if c == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "nil context")
	}

	ctx := sdk.UnwrapSDKContext(c)
	params := k.GetParams(ctx)
	return &types.QueryWhitelistResponse{Whitelist: params.Whitelist}, nil
}
