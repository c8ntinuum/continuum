package keeper

import (
	"context"

	"github.com/cosmos/evm/x/circuit/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) SystemAvailable(c context.Context, _ *types.QuerySystemAvailableRequest) (*types.QuerySystemAvailableResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	available := k.GetSystemAvailable(ctx)

	return &types.QuerySystemAvailableResponse{
		SystemAvailable: available,
	}, nil
}

func (k Keeper) Whitelist(c context.Context, _ *types.QueryWhitelistRequest) (*types.QueryWhitelistResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	params := k.GetParams(ctx)

	return &types.QueryWhitelistResponse{
		Whitelist: params.Whitelist,
	}, nil
}
