package circuit

import (
	"github.com/cosmos/evm/x/circuit/keeper"
	"github.com/cosmos/evm/x/circuit/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	k.SetParams(ctx, genState.Params)
	k.SetSystemAvailable(ctx, genState.State.SystemAvailable)
}

func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	return &types.GenesisState{
		Params: k.GetParams(ctx),
		State: types.CircuitState{
			SystemAvailable: k.GetSystemAvailable(ctx),
		},
	}
}
