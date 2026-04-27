package types

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/stretchr/testify/require"
)

func TestMsgDepositRewardsPoolValidateBasicRejectsZeroAmount(t *testing.T) {
	amount := sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.ZeroInt())
	msg := &MsgDepositRewardsPool{
		Depositor: sdk.AccAddress(make([]byte, 20)).String(),
		Amount:    &amount,
	}

	err := msg.ValidateBasic()
	require.Error(t, err)
	require.Contains(t, err.Error(), "deposit amount must be positive")
}

func TestMsgDepositRewardsPoolValidateBasicRejectsNilAmount(t *testing.T) {
	msg := &MsgDepositRewardsPool{
		Depositor: sdk.AccAddress(make([]byte, 20)).String(),
		Amount:    nil,
	}

	err := msg.ValidateBasic()
	require.Error(t, err)
	require.Contains(t, err.Error(), "amount cannot be nil")
}

func TestMsgClaimRewardsValidateBasicRejectsInvalidRequester(t *testing.T) {
	msg := &MsgClaimRewards{
		ValidatorOperator: sdk.AccAddress(make([]byte, 20)).String(),
		Epoch:             1,
		Requester:         "not-an-address",
	}

	err := msg.ValidateBasic()
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid requester address")
}

func TestMsgClaimRewardsGetSignersUsesRequester(t *testing.T) {
	requester := sdk.AccAddress(bytesRepeat(1, 20)).String()
	validator := sdk.AccAddress(bytesRepeat(2, 20)).String()
	msg := &MsgClaimRewards{
		ValidatorOperator: validator,
		Epoch:             7,
		Requester:         requester,
	}

	signers := msg.GetSigners()
	require.Len(t, signers, 1)
	require.Equal(t, requester, signers[0].String())
}

func bytesRepeat(b byte, size int) []byte {
	out := make([]byte, size)
	for i := range out {
		out[i] = b
	}
	return out
}
