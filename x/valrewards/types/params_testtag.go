//go:build test
// +build test

package types

import (
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

const BLOCKS_IN_EPOCH int64 = 10
const PROPOSER_BONUS_POINTS uint64 = 1
const REWARDS_PER_EPOCH string = "1000000000000000000"

func GetRewardsPerEpoch() sdk.Coin {
	amount, ok := math.NewIntFromString(REWARDS_PER_EPOCH)
	if !ok {
		panic("invalid REWARDS_PER_EPOCH")
	}
	return sdk.NewCoin(evmtypes.DefaultEVMDenom, amount)
}
