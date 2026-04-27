package valrewards

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	vrtypes "github.com/cosmos/evm/x/valrewards/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

func TestValidateOutstandingRewardsFunding(t *testing.T) {
	validAddress := "0x0000000000000000000000000000000000000001"

	tests := []struct {
		name        string
		genesis     vrtypes.GenesisState
		pool        sdk.Coin
		expectError bool
	}{
		{
			name: "funded outstanding rewards",
			genesis: vrtypes.GenesisState{
				ValidatorOutstandingRewards: []vrtypes.GenesisValidatorOutstandingReward{
					{
						Epoch:            1,
						ValidatorAddress: validAddress,
						Amount:           sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(25)),
					},
					{
						Epoch:            2,
						ValidatorAddress: validAddress,
						Amount:           sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(10)),
					},
				},
			},
			pool:        sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(40)),
			expectError: false,
		},
		{
			name: "insufficient pool funding",
			genesis: vrtypes.GenesisState{
				ValidatorOutstandingRewards: []vrtypes.GenesisValidatorOutstandingReward{
					{
						Epoch:            1,
						ValidatorAddress: validAddress,
						Amount:           sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(25)),
					},
					{
						Epoch:            2,
						ValidatorAddress: validAddress,
						Amount:           sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(10)),
					},
				},
			},
			pool:        sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(30)),
			expectError: true,
		},
		{
			name: "foreign denom outstanding reward",
			genesis: vrtypes.GenesisState{
				ValidatorOutstandingRewards: []vrtypes.GenesisValidatorOutstandingReward{
					{
						Epoch:            1,
						ValidatorAddress: validAddress,
						Amount:           sdk.NewCoin("uatom", sdkmath.NewInt(25)),
					},
				},
			},
			pool:        sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(30)),
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateOutstandingRewardsFunding(tc.genesis, tc.pool)
			if tc.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
