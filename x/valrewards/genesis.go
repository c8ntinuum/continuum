package valrewards

import (
	"fmt"

	"github.com/cosmos/evm/x/valrewards/keeper"
	"github.com/cosmos/evm/x/valrewards/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func InitGenesis(ctx sdk.Context, k keeper.Keeper, data types.GenesisState) {
	if err := data.Params.Validate(); err != nil {
		panic(fmt.Errorf("invalid valrewards params: %w", err))
	}
	if err := data.CurrentRewardSettings.Validate(); err != nil {
		panic(fmt.Errorf("invalid current reward settings: %w", err))
	}
	if err := data.NextRewardSettings.Validate(); err != nil {
		panic(fmt.Errorf("invalid next reward settings: %w", err))
	}

	k.SetParams(ctx, data.Params)
	k.SetCurrentRewardSettings(ctx, data.CurrentRewardSettings)
	k.SetNextRewardSettings(ctx, data.NextRewardSettings)
	k.SetEpochState(ctx, data.EpochState)
	k.SetEpochToPay(ctx, data.EpochToPay)

	for _, entry := range data.ValidatorPoints {
		k.SetValidatorRewardPoints(ctx, entry.Epoch, entry.ValidatorAddress, entry.EpochPoints)
	}

	for _, entry := range data.ValidatorOutstandingRewards {
		k.SetValidatorOutstandingReward(ctx, entry.Epoch, entry.ValidatorAddress, entry.Amount)
	}
}

func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	gs := types.DefaultGenesisState()
	gs.Params = k.GetParams(ctx)
	gs.CurrentRewardSettings = k.GetCurrentRewardSettings(ctx)
	gs.NextRewardSettings = k.GetNextRewardSettings(ctx)
	gs.EpochState = k.GetEpochState(ctx)
	gs.EpochToPay = k.GetEpochToPay(ctx)

	k.IterateAllValidatorsPoints(ctx, func(epoch uint64, validatorAddress string, points uint64) bool {
		gs.ValidatorPoints = append(gs.ValidatorPoints, types.GenesisValidatorPoint{
			Epoch:            epoch,
			ValidatorAddress: validatorAddress,
			EpochPoints:      points,
		})
		return false
	})

	k.IterateAllValidatorsOutstandingRewards(ctx, func(epoch uint64, validatorAddress string, amount sdk.Coin) bool {
		if amount.IsZero() {
			return false
		}

		gs.ValidatorOutstandingRewards = append(gs.ValidatorOutstandingRewards, types.GenesisValidatorOutstandingReward{
			Epoch:            epoch,
			ValidatorAddress: validatorAddress,
			Amount:           amount,
		})
		return false
	})

	return gs
}
