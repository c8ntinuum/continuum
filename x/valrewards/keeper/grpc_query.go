package keeper

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	vrtypes "github.com/cosmos/evm/x/valrewards/types"
)

var _ vrtypes.QueryServer = &Keeper{}

func (k Keeper) DelegationRewards(goCtx context.Context, req *vrtypes.QueryDelegationRewardsRequest) (*vrtypes.QueryDelegationRewardsResponse, error) {
	if req == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	delegator, err := vrtypes.ParseAccAddress(req.Delegator)
	if err != nil {
		return nil, err
	}

	maxVals, err := k.stakingKeeper.MaxValidators(ctx)
	if err != nil {
		return nil, err
	}

	rewards, err := k.GetDelegationRewards(ctx, delegator, maxVals, req.Epoch)
	if err != nil {
		return nil, err
	}

	return &vrtypes.QueryDelegationRewardsResponse{Rewards: rewards}, nil
}

func (k Keeper) ValidatorOutstandingRewards(goCtx context.Context, req *vrtypes.QueryValidatorOutstandingRewardsRequest) (*vrtypes.QueryValidatorOutstandingRewardsResponse, error) {
	if req == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "empty request")
	}

	if len([]byte(req.ValidatorAddress)) > 128 {
		return nil, fmt.Errorf("invalid validatorAddress: %s", req.ValidatorAddress)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	rewards := k.GetValidatorOutstandingRewards(ctx, req.Epoch, req.ValidatorAddress)

	return &vrtypes.QueryValidatorOutstandingRewardsResponse{Rewards: rewards}, nil
}

func (k Keeper) RewardsPool(goCtx context.Context, _ *vrtypes.QueryRewardsPoolRequest) (*vrtypes.QueryRewardsPoolResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	pool := k.GetRewardsPool(ctx)
	return &vrtypes.QueryRewardsPoolResponse{Pool: pool}, nil
}
