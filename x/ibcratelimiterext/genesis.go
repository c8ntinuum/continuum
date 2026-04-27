package ibcratelimiterext

import (
	"github.com/cosmos/evm/x/ibcratelimiterext/keeper"
	"github.com/cosmos/evm/x/ibcratelimiterext/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func InitGenesis(ctx sdk.Context, k keeper.Keeper, data types.GenesisState) {
	k.SetParams(ctx, data.Params)
}

func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	return &types.GenesisState{Params: k.GetParams(ctx)}
}
