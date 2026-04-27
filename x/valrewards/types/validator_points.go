package types

import sdk "github.com/cosmos/cosmos-sdk/types"

type EpochValidatorPoints struct {
	ValidatorAddress string
	EpochPoints      uint64
}

type EpochValidatorOutstanding struct {
	ValidatorAddress string
	Amount           sdk.Coin
}
