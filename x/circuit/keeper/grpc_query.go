package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/evm/x/circuit/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) SystemAvailable(c context.Context, req *types.QuerySystemAvailableRequest) (*types.QuerySystemAvailableResponse, error) {
	if req == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	available := k.GetSystemAvailable(ctx)

	return &types.QuerySystemAvailableResponse{
		SystemAvailable: available,
	}, nil
}

func (k Keeper) Whitelist(c context.Context, req *types.QueryWhitelistRequest) (*types.QueryWhitelistResponse, error) {
	if req == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	params := k.GetParams(ctx)

	return &types.QueryWhitelistResponse{
		Whitelist: params.Whitelist,
	}, nil
}
