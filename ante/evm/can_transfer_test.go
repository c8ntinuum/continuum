package evm_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/evm/ante/evm"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
)

func TestCanTransferReturnsErrorForMissingSenderAccount(t *testing.T) {
	keeper := NewExtendedEVMKeeper()
	msg := core.Message{
		From:      common.HexToAddress("0x1000000000000000000000000000000000000001"),
		Value:     big.NewInt(1),
		GasFeeCap: big.NewInt(1),
	}

	err := evm.CanTransfer(sdk.Context{}, keeper, msg, big.NewInt(0), evmtypes.DefaultParams(), false)
	require.Error(t, err)
	require.ErrorIs(t, err, errortypes.ErrInsufficientFunds)
}
