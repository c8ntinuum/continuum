package cosmos

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/cosmos/evm/msd"
)

type MinSelfDelegationDecorator struct {
	sk *stakingkeeper.Keeper
}

func NewMinSelfDelegationDecorator(stakingKeeper *stakingkeeper.Keeper) MinSelfDelegationDecorator {
	return MinSelfDelegationDecorator{
		sk: stakingKeeper,
	}
}

func (msdd MinSelfDelegationDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// Check all messages in the transaction
	err := msd.CheckMSDInTx(ctx, tx)
	if err != nil {
		return ctx, errorsmod.Wrap(msd.ErrMSD, err.Error())
	}
	// Continue to next ante handler if all checks pass
	return next(ctx, tx, simulate)
}
