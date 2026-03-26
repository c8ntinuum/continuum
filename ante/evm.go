package ante

import (
	cosmosante "github.com/cosmos/evm/ante/cosmos"
	evmante "github.com/cosmos/evm/ante/evm"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// newMonoEVMAnteHandler creates the sdk.AnteHandler implementation for the EVM transactions.
func newMonoEVMAnteHandler(ctx sdk.Context, options HandlerOptions) sdk.AnteHandler {
	evmParams := options.EvmKeeper.GetParams(ctx)
	feemarketParams := options.FeeMarketKeeper.GetParams(ctx)
	decorators := []sdk.AnteDecorator{
		cosmosante.NewCircuitAvailableDecorator(options.CircuitKeeper),
		evmante.NewEVMMonoDecorator(
			options.AccountKeeper,
			options.FeeMarketKeeper,
			options.EvmKeeper,
			options.MaxTxGasWanted,
			&evmParams,
			&feemarketParams,
		),
		NewTxListenerDecorator(options.PendingTxListener),
	}

	return sdk.ChainAnteDecorators(decorators...)
}
