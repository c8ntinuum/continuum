package keeper

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	vrtypes "github.com/cosmos/evm/x/valrewards/types"
)

var _ vrtypes.MsgServer = &Keeper{}

func (k Keeper) ClaimRewards(goCtx context.Context, msg *vrtypes.MsgClaimRewards) (*vrtypes.MsgClaimRewardsResponse, error) {
	if msg == nil {
		return nil, errorsmod.Wrap(errortypes.ErrInvalidRequest, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	delegator, err := vrtypes.ParseAccAddress(msg.Delegator)
	if err != nil {
		return nil, err
	}

	maxVals, err := k.stakingKeeper.MaxValidators(ctx)
	if err != nil {
		return nil, err
	}
	if msg.MaxRetrieve > maxVals {
		return nil, fmt.Errorf("maxRetrieve (%d) parameter exceeds the maximum number of validators (%d)", msg.MaxRetrieve, maxVals)
	}

	if _, err := k.ClaimDelegationRewards(ctx, delegator, msg.MaxRetrieve, msg.Epoch); err != nil {
		return nil, err
	}

	return &vrtypes.MsgClaimRewardsResponse{}, nil
}

func (k Keeper) DepositRewardsPool(goCtx context.Context, msg *vrtypes.MsgDepositRewardsPool) (*vrtypes.MsgDepositRewardsPoolResponse, error) {
	if msg == nil {
		return nil, errorsmod.Wrap(errortypes.ErrInvalidRequest, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	depositor, err := vrtypes.ParseAccAddress(msg.Depositor)
	if err != nil {
		return nil, err
	}

	if msg.Amount == nil {
		return nil, errortypes.ErrInvalidCoins
	}
	if msg.Amount.Denom != evmtypes.DefaultEVMDenom {
		return nil, fmt.Errorf("invalid amount denom: %s", msg.Amount.Denom)
	}

	if err := k.DepositRewardsPoolCoins(ctx, depositor, *msg.Amount); err != nil {
		return nil, err
	}

	return &vrtypes.MsgDepositRewardsPoolResponse{}, nil
}
