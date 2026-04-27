package types

import (
	"fmt"
	"strings"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

type GenesisValidatorPoint struct {
	Epoch            uint64 `json:"epoch"`
	ValidatorAddress string `json:"validator_address"`
	EpochPoints      uint64 `json:"epoch_points"`
}

type GenesisValidatorOutstandingReward struct {
	Epoch            uint64   `json:"epoch"`
	ValidatorAddress string   `json:"validator_address"`
	Amount           sdk.Coin `json:"amount"`
}

type GenesisState struct {
	Params                      Params                              `json:"params"`
	CurrentRewardSettings       RewardSettings                      `json:"current_reward_settings"`
	NextRewardSettings          RewardSettings                      `json:"next_reward_settings"`
	EpochState                  EpochState                          `json:"epoch_state"`
	EpochToPay                  uint64                              `json:"epoch_to_pay"`
	ValidatorPoints             []GenesisValidatorPoint             `json:"validator_points"`
	ValidatorOutstandingRewards []GenesisValidatorOutstandingReward `json:"validator_outstanding_rewards"`
}

func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:                      DefaultParams(),
		CurrentRewardSettings:       DefaultRewardSettings(),
		NextRewardSettings:          DefaultRewardSettings(),
		EpochState:                  DefaultEpochState(),
		EpochToPay:                  0,
		ValidatorPoints:             []GenesisValidatorPoint{},
		ValidatorOutstandingRewards: []GenesisValidatorOutstandingReward{},
	}
}

func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	if err := gs.CurrentRewardSettings.Validate(); err != nil {
		return errorsmod.Wrap(err, "invalid current_reward_settings")
	}
	if err := gs.NextRewardSettings.Validate(); err != nil {
		return errorsmod.Wrap(err, "invalid next_reward_settings")
	}
	if gs.EpochState.BlocksIntoCurrentEpoch < 0 {
		return fmt.Errorf("epoch_state.blocks_into_current_epoch cannot be negative")
	}
	if gs.EpochState.BlocksIntoCurrentEpoch >= gs.CurrentRewardSettings.BlocksInEpoch {
		return fmt.Errorf("epoch_state.blocks_into_current_epoch must be smaller than current_reward_settings.blocks_in_epoch")
	}
	if gs.EpochToPay > gs.EpochState.CurrentEpoch {
		return fmt.Errorf("epoch_to_pay cannot exceed epoch_state.current_epoch")
	}

	pointEntries := make(map[string]struct{}, len(gs.ValidatorPoints))
	for _, entry := range gs.ValidatorPoints {
		if entry.Epoch > gs.EpochState.CurrentEpoch {
			return fmt.Errorf("validator points entry epoch %d cannot exceed epoch_state.current_epoch %d", entry.Epoch, gs.EpochState.CurrentEpoch)
		}
		if err := ValidateValidatorOperatorAddress(entry.ValidatorAddress); err != nil {
			return errorsmod.Wrap(err, "invalid validator points address")
		}

		key := fmt.Sprintf("%d/%s", entry.Epoch, entry.ValidatorAddress)
		if _, exists := pointEntries[key]; exists {
			return fmt.Errorf("duplicate validator points entry for epoch %d and validator %s", entry.Epoch, entry.ValidatorAddress)
		}
		pointEntries[key] = struct{}{}
	}

	outstandingEntries := make(map[string]struct{}, len(gs.ValidatorOutstandingRewards))
	for _, entry := range gs.ValidatorOutstandingRewards {
		if entry.Epoch > gs.EpochState.CurrentEpoch {
			return fmt.Errorf("validator outstanding rewards entry epoch %d cannot exceed epoch_state.current_epoch %d", entry.Epoch, gs.EpochState.CurrentEpoch)
		}
		if err := ValidateValidatorOperatorAddress(entry.ValidatorAddress); err != nil {
			return errorsmod.Wrap(err, "invalid validator outstanding rewards address")
		}
		if !entry.Amount.IsValid() {
			return errorsmod.Wrapf(errortypes.ErrInvalidCoins, "invalid outstanding reward coin %s", entry.Amount.String())
		}
		if entry.Amount.Denom != evmtypes.DefaultEVMDenom {
			return fmt.Errorf(
				"invalid outstanding reward denom for epoch %d and validator %s: got %s, expected %s",
				entry.Epoch,
				entry.ValidatorAddress,
				entry.Amount.Denom,
				evmtypes.DefaultEVMDenom,
			)
		}

		key := fmt.Sprintf("%d/%s", entry.Epoch, entry.ValidatorAddress)
		if _, exists := outstandingEntries[key]; exists {
			return fmt.Errorf("duplicate validator outstanding rewards entry for epoch %d and validator %s", entry.Epoch, entry.ValidatorAddress)
		}
		outstandingEntries[key] = struct{}{}
	}

	return nil
}

func ValidateValidatorOperatorAddress(addr string) error {
	if strings.TrimSpace(addr) == "" {
		return errortypes.ErrInvalidAddress
	}

	_, err := sdk.ValAddressFromBech32(addr)
	return err
}
