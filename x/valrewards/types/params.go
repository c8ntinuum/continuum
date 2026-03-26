//go:build !test
// +build !test

package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

const BLOCKS_IN_EPOCH int64 = 17280
const PROPOSER_BONUS_POINTS uint64 = 1
const REWARDS_PER_EPOCH string = "45004521205479450000000"

func GetRewardsPerEpoch() sdk.Coin {
	amount, ok := math.NewIntFromString(REWARDS_PER_EPOCH)
	if !ok {
		panic("invalid big int literal for HugeRewardCoin")
	}
	return sdk.NewCoin(evmtypes.DefaultEVMDenom, amount)
}
