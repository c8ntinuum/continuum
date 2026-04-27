package cosmos

import (
	errorsmod "cosmossdk.io/errors"

	anteinterfaces "github.com/cosmos/evm/ante/interfaces"
	circuittype "github.com/cosmos/evm/x/circuit/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
)

type CircuitAvailableDecorator struct {
	circuitKeeper anteinterfaces.CircuitKeeper
}

func NewCircuitAvailableDecorator(circuitKeeper anteinterfaces.CircuitKeeper) CircuitAvailableDecorator {
	return CircuitAvailableDecorator{circuitKeeper: circuitKeeper}
}

func (d CircuitAvailableDecorator) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, sim bool, next sdk.AnteHandler,
) (sdk.Context, error) {
	if d.circuitKeeper.GetSystemAvailable(ctx) {
		return next(ctx, tx, sim)
	}

	msgs := tx.GetMsgs()
	for _, msg := range msgs {
		if _, ok := msg.(*circuittype.MsgUpdateCircuit); !ok {
			return ctx, errorsmod.Wrap(errortypes.ErrUnauthorized, "system unavailable")
		}
	}

	return next(ctx, tx, sim)
}
