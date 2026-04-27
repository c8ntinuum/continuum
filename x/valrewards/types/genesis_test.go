package types

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/stretchr/testify/require"
)

func TestGenesisStateValidate(t *testing.T) {
	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount("cosmos", "cosmospub")
	cfg.SetBech32PrefixForValidator("cosmosvaloper", "cosmosvaloperpub")
	cfg.SetBech32PrefixForConsensusNode("cosmosvalcons", "cosmosvalconspub")

	validAddress := sdk.ValAddress([]byte("valid_validator_addr")).String()
	validWhitelistAddress := sdk.AccAddress([]byte("valid_whitelist_address")).String()

	tests := []struct {
		name        string
		genesis     GenesisState
		expectError bool
	}{
		{
			name:        "default genesis",
			genesis:     *DefaultGenesisState(),
			expectError: false,
		},
		{
			name: "valid populated genesis",
			genesis: GenesisState{
				Params: Params{
					Whitelist: []string{validWhitelistAddress},
				},
				CurrentRewardSettings: RewardSettings{
					BlocksInEpoch:   100,
					RewardsPerEpoch: "1000000000000000000",
					RewardingPaused: false,
				},
				NextRewardSettings: RewardSettings{
					BlocksInEpoch:   120,
					RewardsPerEpoch: "2000000000000000000",
					RewardingPaused: true,
				},
				EpochState: EpochState{
					CurrentEpoch:           4,
					BlocksIntoCurrentEpoch: 10,
				},
				EpochToPay: 4,
				ValidatorPoints: []GenesisValidatorPoint{
					{Epoch: 1, ValidatorAddress: validAddress, EpochPoints: 10},
				},
				ValidatorOutstandingRewards: []GenesisValidatorOutstandingReward{
					{Epoch: 1, ValidatorAddress: validAddress, Amount: sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(25))},
				},
			},
			expectError: false,
		},
		{
			name: "duplicate validator points entry",
			genesis: GenesisState{
				Params:                DefaultParams(),
				CurrentRewardSettings: DefaultRewardSettings(),
				NextRewardSettings:    DefaultRewardSettings(),
				EpochState: EpochState{
					CurrentEpoch:           1,
					BlocksIntoCurrentEpoch: 0,
				},
				ValidatorPoints: []GenesisValidatorPoint{
					{Epoch: 1, ValidatorAddress: validAddress, EpochPoints: 10},
					{Epoch: 1, ValidatorAddress: validAddress, EpochPoints: 20},
				},
			},
			expectError: true,
		},
		{
			name: "duplicate outstanding rewards entry",
			genesis: GenesisState{
				Params:                DefaultParams(),
				CurrentRewardSettings: DefaultRewardSettings(),
				NextRewardSettings:    DefaultRewardSettings(),
				EpochState: EpochState{
					CurrentEpoch:           1,
					BlocksIntoCurrentEpoch: 0,
				},
				ValidatorOutstandingRewards: []GenesisValidatorOutstandingReward{
					{Epoch: 1, ValidatorAddress: validAddress, Amount: sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(25))},
					{Epoch: 1, ValidatorAddress: validAddress, Amount: sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(30))},
				},
			},
			expectError: true,
		},
		{
			name: "invalid validator address",
			genesis: GenesisState{
				Params:                DefaultParams(),
				CurrentRewardSettings: DefaultRewardSettings(),
				NextRewardSettings:    DefaultRewardSettings(),
				EpochState: EpochState{
					CurrentEpoch:           1,
					BlocksIntoCurrentEpoch: 0,
				},
				ValidatorPoints: []GenesisValidatorPoint{
					{Epoch: 1, ValidatorAddress: "not-an-address", EpochPoints: 10},
				},
			},
			expectError: true,
		},
		{
			name: "hex validator address not allowed",
			genesis: GenesisState{
				Params:                DefaultParams(),
				CurrentRewardSettings: DefaultRewardSettings(),
				NextRewardSettings:    DefaultRewardSettings(),
				EpochState: EpochState{
					CurrentEpoch:           1,
					BlocksIntoCurrentEpoch: 0,
				},
				ValidatorPoints: []GenesisValidatorPoint{
					{Epoch: 1, ValidatorAddress: "0x0000000000000000000000000000000000000001", EpochPoints: 10},
				},
			},
			expectError: true,
		},
		{
			name: "invalid outstanding reward coin",
			genesis: GenesisState{
				Params:                DefaultParams(),
				CurrentRewardSettings: DefaultRewardSettings(),
				NextRewardSettings:    DefaultRewardSettings(),
				EpochState: EpochState{
					CurrentEpoch:           1,
					BlocksIntoCurrentEpoch: 0,
				},
				ValidatorOutstandingRewards: []GenesisValidatorOutstandingReward{
					{
						Epoch:            1,
						ValidatorAddress: validAddress,
						Amount: sdk.Coin{
							Denom:  "ctm",
							Amount: sdkmath.NewInt(-1),
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "invalid outstanding reward denom",
			genesis: GenesisState{
				Params:                DefaultParams(),
				CurrentRewardSettings: DefaultRewardSettings(),
				NextRewardSettings:    DefaultRewardSettings(),
				EpochState: EpochState{
					CurrentEpoch:           1,
					BlocksIntoCurrentEpoch: 0,
				},
				ValidatorOutstandingRewards: []GenesisValidatorOutstandingReward{
					{
						Epoch:            1,
						ValidatorAddress: validAddress,
						Amount:           sdk.NewCoin("uatom", sdkmath.NewInt(25)),
					},
				},
			},
			expectError: true,
		},
		{
			name: "validator points entry in future epoch",
			genesis: GenesisState{
				Params:                DefaultParams(),
				CurrentRewardSettings: DefaultRewardSettings(),
				NextRewardSettings:    DefaultRewardSettings(),
				EpochState: EpochState{
					CurrentEpoch:           1,
					BlocksIntoCurrentEpoch: 0,
				},
				ValidatorPoints: []GenesisValidatorPoint{
					{Epoch: 2, ValidatorAddress: validAddress, EpochPoints: 10},
				},
			},
			expectError: true,
		},
		{
			name: "outstanding rewards entry in future epoch",
			genesis: GenesisState{
				Params:                DefaultParams(),
				CurrentRewardSettings: DefaultRewardSettings(),
				NextRewardSettings:    DefaultRewardSettings(),
				EpochState: EpochState{
					CurrentEpoch:           1,
					BlocksIntoCurrentEpoch: 0,
				},
				ValidatorOutstandingRewards: []GenesisValidatorOutstandingReward{
					{
						Epoch:            2,
						ValidatorAddress: validAddress,
						Amount:           sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(25)),
					},
				},
			},
			expectError: true,
		},
		{
			name: "invalid current reward settings",
			genesis: GenesisState{
				Params: DefaultParams(),
				CurrentRewardSettings: RewardSettings{
					BlocksInEpoch:   10,
					RewardsPerEpoch: "1000000000000000000",
				},
				NextRewardSettings: DefaultRewardSettings(),
				EpochState:         DefaultEpochState(),
			},
			expectError: true,
		},
		{
			name: "invalid epoch state progression",
			genesis: GenesisState{
				Params:                DefaultParams(),
				CurrentRewardSettings: DefaultRewardSettings(),
				NextRewardSettings:    DefaultRewardSettings(),
				EpochState: EpochState{
					CurrentEpoch:           1,
					BlocksIntoCurrentEpoch: DefaultRewardSettings().BlocksInEpoch,
				},
			},
			expectError: true,
		},
		{
			name: "epoch_to_pay exceeds current epoch",
			genesis: GenesisState{
				Params:                DefaultParams(),
				CurrentRewardSettings: DefaultRewardSettings(),
				NextRewardSettings:    DefaultRewardSettings(),
				EpochState: EpochState{
					CurrentEpoch:           1,
					BlocksIntoCurrentEpoch: 0,
				},
				EpochToPay: 2,
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.genesis.Validate()
			if tc.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
