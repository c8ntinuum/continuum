package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestGenesisStateValidate(t *testing.T) {
	operator := sdk.AccAddress([]byte("operator_1")).String()

	testCases := []struct {
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
			name: "offline with empty whitelist is invalid",
			genesis: GenesisState{
				Params: Params{
					Whitelist: []string{},
				},
				State: CircuitState{
					SystemAvailable: false,
				},
			},
			expectError: true,
		},
		{
			name: "offline with non-empty whitelist is valid",
			genesis: GenesisState{
				Params: Params{
					Whitelist: []string{operator},
				},
				State: CircuitState{
					SystemAvailable: false,
				},
			},
			expectError: false,
		},
		{
			name: "online with empty whitelist is valid",
			genesis: GenesisState{
				Params: Params{
					Whitelist: []string{},
				},
				State: CircuitState{
					SystemAvailable: true,
				},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
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
