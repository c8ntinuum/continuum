package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

const BLOCKS_IN_EPOCH int64 = 17280                        // Blocks per day
const PROPOSER_BONUS_POINTS uint64 = 1                     // Bonus points per proposer
const REWARDS_PER_EPOCH string = "45004521205479450000000" // Reawards per day // 45_004,521,205,479,450,000,000 CTM

// const BLOCKS_IN_EPOCH int64 = 10                          // Blocks per day
// const PROPOSER_BONUS_POINTS uint64 = 10                   // Bonus points per proposer
// const REWARDS_PER_EPOCH string = "1000000000000000000000" // Reawards per day 1_000

func GetRewardsPerEpoch() sdk.Coin {
	amount, ok := math.NewIntFromString(REWARDS_PER_EPOCH)
	if !ok {
		panic("invalid big int literal for HugeRewardCoin")
	}
	return sdk.NewCoin(evmtypes.DefaultEVMDenom, amount)
}
