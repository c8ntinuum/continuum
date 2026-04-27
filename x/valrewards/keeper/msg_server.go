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

func (k Keeper) UpdateParams(goCtx context.Context, msg *vrtypes.MsgUpdateParams) (*vrtypes.MsgUpdateParamsResponse, error) {
	if msg == nil {
		return nil, errorsmod.Wrap(errortypes.ErrInvalidRequest, "empty request")
	}
	authority, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return nil, errorsmod.Wrap(err, "invalid authority address")
	}
	if !k.authority.Equals(authority) {
		return nil, errorsmod.Wrap(errortypes.ErrUnauthorized, "invalid authority")
	}
	if msg.Params == nil {
		return nil, errorsmod.Wrap(errortypes.ErrInvalidRequest, "empty params")
	}
	if err := msg.Params.Validate(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	k.SetParams(ctx, *msg.Params)
	return &vrtypes.MsgUpdateParamsResponse{}, nil
}

func (k Keeper) SetBlocksInEpoch(goCtx context.Context, msg *vrtypes.MsgSetBlocksInEpoch) (*vrtypes.MsgSetBlocksInEpochResponse, error) {
	if msg == nil {
		return nil, errorsmod.Wrap(errortypes.ErrInvalidRequest, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return nil, errorsmod.Wrap(err, "invalid signer address")
	}
	if !k.IsWhitelisted(ctx, signer) {
		return nil, errorsmod.Wrap(errortypes.ErrUnauthorized, "signer not whitelisted")
	}

	settings := k.GetNextRewardSettings(ctx)
	settings.BlocksInEpoch = msg.BlocksInEpoch
	if err := settings.Validate(); err != nil {
		return nil, err
	}

	k.SetNextRewardSettings(ctx, settings)
	return &vrtypes.MsgSetBlocksInEpochResponse{}, nil
}

func (k Keeper) SetRewardsPerEpoch(goCtx context.Context, msg *vrtypes.MsgSetRewardsPerEpoch) (*vrtypes.MsgSetRewardsPerEpochResponse, error) {
	if msg == nil {
		return nil, errorsmod.Wrap(errortypes.ErrInvalidRequest, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return nil, errorsmod.Wrap(err, "invalid signer address")
	}
	if !k.IsWhitelisted(ctx, signer) {
		return nil, errorsmod.Wrap(errortypes.ErrUnauthorized, "signer not whitelisted")
	}

	settings := k.GetNextRewardSettings(ctx)
	settings.RewardsPerEpoch = msg.RewardsPerEpoch
	if err := settings.Validate(); err != nil {
		return nil, err
	}

	k.SetNextRewardSettings(ctx, settings)
	return &vrtypes.MsgSetRewardsPerEpochResponse{}, nil
}

func (k Keeper) SetRewardingPaused(goCtx context.Context, msg *vrtypes.MsgSetRewardingPaused) (*vrtypes.MsgSetRewardingPausedResponse, error) {
	if msg == nil {
		return nil, errorsmod.Wrap(errortypes.ErrInvalidRequest, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return nil, errorsmod.Wrap(err, "invalid signer address")
	}
	if !k.IsWhitelisted(ctx, signer) {
		return nil, errorsmod.Wrap(errortypes.ErrUnauthorized, "signer not whitelisted")
	}

	settings := k.GetNextRewardSettings(ctx)
	settings.RewardingPaused = msg.RewardingPaused
	if err := settings.Validate(); err != nil {
		return nil, err
	}

	k.SetNextRewardSettings(ctx, settings)
	return &vrtypes.MsgSetRewardingPausedResponse{}, nil
}

func (k Keeper) ClaimRewards(goCtx context.Context, msg *vrtypes.MsgClaimRewards) (*vrtypes.MsgClaimRewardsResponse, error) {
	if msg == nil {
		return nil, errorsmod.Wrap(errortypes.ErrInvalidRequest, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	operator, err := vrtypes.ParseAccAddress(msg.ValidatorOperator)
	if err != nil {
		return nil, err
	}
	if _, err := vrtypes.ParseAccAddress(msg.Requester); err != nil {
		return nil, err
	}

	if _, err := k.ClaimOperatorRewards(ctx, operator, msg.Epoch); err != nil {
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
