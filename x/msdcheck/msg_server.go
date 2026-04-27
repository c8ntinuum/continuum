package msdcheck

import (
	"context"
	"math/big"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

const minSelfDelegationStr = "888888000000000000000000"

var minSelfDelegation = func() math.Int {
	bi, ok := new(big.Int).SetString(minSelfDelegationStr, 10)
	if !ok {
		panic("invalid minSelfDelegationStr constant")
	}
	return math.NewIntFromBigInt(bi)
}()

type MsgServer struct {
	stakingtypes.MsgServer
	keeper *stakingkeeper.Keeper
}

func NewMsgServer(base stakingtypes.MsgServer, keeper *stakingkeeper.Keeper) stakingtypes.MsgServer {
	return &MsgServer{
		MsgServer: base,
		keeper:    keeper,
	}
}

func (m *MsgServer) CreateValidator(ctx context.Context, msg *stakingtypes.MsgCreateValidator) (*stakingtypes.MsgCreateValidatorResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	bondDenom, err := m.keeper.BondDenom(sdkCtx)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, err.Error())
	}

	if msg.Value.Denom != bondDenom {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "initial self-delegation must be in bond denom %s", bondDenom)
	}

	if msg.Value.Amount.LT(minSelfDelegation) {
		return nil, errorsmod.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"initial self-delegation amount %s < minimum %s",
			msg.Value.Amount.String(),
			minSelfDelegation.String(),
		)
	}

	if msg.MinSelfDelegation.LT(minSelfDelegation) {
		return nil, errorsmod.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"initial self-delegation specified %s < minimum %s",
			msg.MinSelfDelegation.String(),
			minSelfDelegation.String(),
		)
	}

	return m.MsgServer.CreateValidator(ctx, msg)
}

func (m *MsgServer) EditValidator(ctx context.Context, msg *stakingtypes.MsgEditValidator) (*stakingtypes.MsgEditValidatorResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	valAddr, err := sdk.ValAddressFromBech32(msg.ValidatorAddress)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, err.Error())
	}

	if _, err := m.keeper.GetValidator(sdkCtx, valAddr); err != nil {
		return nil, err
	}

	if msg.MinSelfDelegation != nil && msg.MinSelfDelegation.LT(minSelfDelegation) {
		return nil, errorsmod.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"proposed min self-delegation %s < minimum %s",
			msg.MinSelfDelegation.String(),
			minSelfDelegation.String(),
		)
	}

	return m.MsgServer.EditValidator(ctx, msg)
}

var _ stakingtypes.MsgServer = (*MsgServer)(nil)
