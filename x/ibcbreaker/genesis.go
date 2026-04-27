package ibcbreaker

import (
	"github.com/cosmos/evm/x/ibcbreaker/keeper"
	"github.com/cosmos/evm/x/ibcbreaker/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	k.SetParams(ctx, genState.Params)
	k.SetIbcAvailable(ctx, genState.State.IbcAvailable)
}

func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	return &types.GenesisState{
		Params: k.GetParams(ctx),
		State: types.IbcBreakerState{
			IbcAvailable: k.GetIbcAvailable(ctx),
		},
	}
}
