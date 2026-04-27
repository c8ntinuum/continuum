package types

import (
	"fmt"
	"strings"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

const (
	MinBlocksInEpoch int64 = 20
	MaxBlocksInEpoch int64 = 6_500_000

	MinRewardsPerEpoch string = "1000000000000000000"
	MaxRewardsPerEpoch string = "25000000000000000000000000"
)

func DefaultParams() Params {
	return Params{Whitelist: []string{}}
}

func DefaultRewardSettings() RewardSettings {
	return RewardSettings{
		BlocksInEpoch:   BLOCKS_IN_EPOCH,
		RewardsPerEpoch: REWARDS_PER_EPOCH,
		RewardingPaused: REWARDING_PAUSED,
	}
}

func DefaultEpochState() EpochState {
	return EpochState{
		CurrentEpoch:           0,
		BlocksIntoCurrentEpoch: 0,
	}
}

func GetRewardsPerEpoch() sdk.Coin {
	coin, err := DefaultRewardSettings().RewardsCoin()
	if err != nil {
		panic(err)
	}
	return coin
}

func ValidateBlocksInEpoch(blocksInEpoch int64) error {
	if blocksInEpoch < MinBlocksInEpoch {
		return fmt.Errorf("blocks_in_epoch must be at least %d", MinBlocksInEpoch)
	}
	if blocksInEpoch > MaxBlocksInEpoch {
		return fmt.Errorf("blocks_in_epoch must be at most %d", MaxBlocksInEpoch)
	}
	return nil
}

func ValidateRewardsPerEpoch(rewardsPerEpoch string) error {
	_, err := ParseRewardsPerEpoch(rewardsPerEpoch)
	return err
}

func ParseRewardsPerEpoch(rewardsPerEpoch string) (sdkmath.Int, error) {
	if strings.TrimSpace(rewardsPerEpoch) == "" {
		return sdkmath.Int{}, fmt.Errorf("rewards_per_epoch cannot be empty")
	}

	amount, ok := sdkmath.NewIntFromString(rewardsPerEpoch)
	if !ok {
		return sdkmath.Int{}, fmt.Errorf("invalid rewards_per_epoch %q", rewardsPerEpoch)
	}

	minAmount, ok := sdkmath.NewIntFromString(MinRewardsPerEpoch)
	if !ok {
		panic("invalid MinRewardsPerEpoch")
	}
	maxAmount, ok := sdkmath.NewIntFromString(MaxRewardsPerEpoch)
	if !ok {
		panic("invalid MaxRewardsPerEpoch")
	}

	if amount.LT(minAmount) {
		return sdkmath.Int{}, fmt.Errorf("rewards_per_epoch must be at least %s", MinRewardsPerEpoch)
	}
	if amount.GT(maxAmount) {
		return sdkmath.Int{}, fmt.Errorf("rewards_per_epoch must be at most %s", MaxRewardsPerEpoch)
	}

	return amount, nil
}

func (p Params) Validate() error {
	seen := make(map[string]struct{}, len(p.Whitelist))
	for _, addr := range p.Whitelist {
		accAddr, err := sdk.AccAddressFromBech32(addr)
		if err != nil {
			return fmt.Errorf("invalid whitelist address %q: %w", addr, err)
		}
		normalized := accAddr.String()
		if _, ok := seen[normalized]; ok {
			return fmt.Errorf("duplicate whitelist address %q", normalized)
		}
		seen[normalized] = struct{}{}
	}

	return nil
}

func (s RewardSettings) Validate() error {
	if err := ValidateBlocksInEpoch(s.BlocksInEpoch); err != nil {
		return err
	}
	if err := ValidateRewardsPerEpoch(s.RewardsPerEpoch); err != nil {
		return err
	}

	return nil
}

func (s RewardSettings) RewardsCoin() (sdk.Coin, error) {
	amount, err := ParseRewardsPerEpoch(s.RewardsPerEpoch)
	if err != nil {
		return sdk.Coin{}, err
	}

	return sdk.NewCoin(evmtypes.DefaultEVMDenom, amount), nil
}
