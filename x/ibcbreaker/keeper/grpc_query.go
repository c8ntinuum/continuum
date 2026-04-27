package keeper

import (
	"context"

	"github.com/cosmos/evm/x/ibcbreaker/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) IbcAvailable(c context.Context, _ *types.QueryIbcAvailableRequest) (*types.QueryIbcAvailableResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	available := k.GetIbcAvailable(ctx)

	return &types.QueryIbcAvailableResponse{
		IbcAvailable: available,
	}, nil
}

func (k Keeper) Whitelist(c context.Context, _ *types.QueryWhitelistRequest) (*types.QueryWhitelistResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	params := k.GetParams(ctx)

	return &types.QueryWhitelistResponse{
		Whitelist: params.Whitelist,
	}, nil
}
