package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestRewardSettingsValidate(t *testing.T) {
	tests := []struct {
		name        string
		settings    RewardSettings
		expectError bool
	}{
		{
			name: "valid settings",
			settings: RewardSettings{
				BlocksInEpoch:   20,
				RewardsPerEpoch: MinRewardsPerEpoch,
				RewardingPaused: false,
			},
		},
		{
			name: "blocks too small",
			settings: RewardSettings{
				BlocksInEpoch:   19,
				RewardsPerEpoch: MinRewardsPerEpoch,
			},
			expectError: true,
		},
		{
			name: "blocks too large",
			settings: RewardSettings{
				BlocksInEpoch:   MaxBlocksInEpoch + 1,
				RewardsPerEpoch: MinRewardsPerEpoch,
			},
			expectError: true,
		},
		{
			name: "empty rewards",
			settings: RewardSettings{
				BlocksInEpoch:   20,
				RewardsPerEpoch: "",
			},
			expectError: true,
		},
		{
			name: "non numeric rewards",
			settings: RewardSettings{
				BlocksInEpoch:   20,
				RewardsPerEpoch: "abc",
			},
			expectError: true,
		},
		{
			name: "rewards below minimum",
			settings: RewardSettings{
				BlocksInEpoch:   20,
				RewardsPerEpoch: "999999999999999999",
			},
			expectError: true,
		},
		{
			name: "rewards above maximum",
			settings: RewardSettings{
				BlocksInEpoch:   20,
				RewardsPerEpoch: "25000000000000000000000001",
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.settings.Validate()
			if tc.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestParamsValidate(t *testing.T) {
	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount("cosmos", "cosmospub")
	addr := sdk.AccAddress([]byte("whitelist_address__1")).String()
	require.NoError(t, Params{Whitelist: []string{addr}}.Validate())
	require.Error(t, Params{Whitelist: []string{"bad"}}.Validate())
	require.Error(t, Params{Whitelist: []string{addr, addr}}.Validate())
}
