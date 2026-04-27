package types

import (
	"fmt"
	"testing"

	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/require"
)

func TestWithReservedPrecompilesRegistersAllReservedSlots(t *testing.T) {
	precompiles := NewStaticPrecompiles().WithReservedPrecompiles()

	expectedAddresses := []string{
		evmtypes.ReservedSlot15PrecompileAddress,
		evmtypes.ReservedSlot16PrecompileAddress,
		evmtypes.ReservedSlot17PrecompileAddress,
		evmtypes.ReservedSlot18PrecompileAddress,
		evmtypes.ReservedSlot19PrecompileAddress,
		evmtypes.ReservedSlot20PrecompileAddress,
		evmtypes.ReservedSlot21PrecompileAddress,
		evmtypes.ReservedSlot22PrecompileAddress,
		evmtypes.ReservedSlot23PrecompileAddress,
		evmtypes.ReservedSlot24PrecompileAddress,
		evmtypes.ReservedSlot25PrecompileAddress,
		evmtypes.ReservedSlot26PrecompileAddress,
		evmtypes.ReservedSlot27PrecompileAddress,
		evmtypes.ReservedSlot28PrecompileAddress,
		evmtypes.ReservedSlot29PrecompileAddress,
		evmtypes.ReservedSlot30PrecompileAddress,
		evmtypes.ReservedSlot31PrecompileAddress,
		evmtypes.ReservedSlot32PrecompileAddress,
		evmtypes.ReservedSlot33PrecompileAddress,
		evmtypes.ReservedSlot34PrecompileAddress,
		evmtypes.ReservedSlot35PrecompileAddress,
		evmtypes.ReservedSlot36PrecompileAddress,
		evmtypes.ReservedSlot37PrecompileAddress,
		evmtypes.ReservedSlot38PrecompileAddress,
		evmtypes.ReservedSlot39PrecompileAddress,
		evmtypes.ReservedSlot40PrecompileAddress,
		evmtypes.ReservedSlot41PrecompileAddress,
		evmtypes.ReservedSlot42PrecompileAddress,
		evmtypes.ReservedSlot43PrecompileAddress,
		evmtypes.ReservedSlot44PrecompileAddress,
		evmtypes.ReservedSlot45PrecompileAddress,
		evmtypes.ReservedSlot46PrecompileAddress,
		evmtypes.ReservedSlot47PrecompileAddress,
		evmtypes.ReservedSlot48PrecompileAddress,
		evmtypes.ReservedSlot49PrecompileAddress,
		evmtypes.ReservedSlot50PrecompileAddress,
	}

	require.Len(t, precompiles, len(expectedAddresses))

	_, found := precompiles[common.Address{}]
	require.False(t, found)

	for _, addressHex := range expectedAddresses {
		_, found := precompiles[common.HexToAddress(addressHex)]
		require.Truef(t, found, "reserved precompile not registered for %s", addressHex)
	}
}

func TestAssertAvailableStaticPrecompilesRegistered(t *testing.T) {
	precompiles := make(map[common.Address]vm.PrecompiledContract, len(evmtypes.AvailableStaticPrecompiles))
	for _, addressHex := range evmtypes.AvailableStaticPrecompiles {
		precompiles[common.HexToAddress(addressHex)] = noopPrecompile{}
	}

	require.NotPanics(t, func() {
		assertAvailableStaticPrecompilesRegistered(precompiles)
	})
}

func TestAssertAvailableStaticPrecompilesRegisteredPanicsOnMissingAddress(t *testing.T) {
	missingAddress := evmtypes.BankPrecompileAddress
	precompiles := make(map[common.Address]vm.PrecompiledContract, len(evmtypes.AvailableStaticPrecompiles)-1)
	for _, addressHex := range evmtypes.AvailableStaticPrecompiles {
		if addressHex == missingAddress {
			continue
		}
		precompiles[common.HexToAddress(addressHex)] = noopPrecompile{}
	}

	require.PanicsWithError(t, fmt.Sprintf("available static precompile %s is not registered", missingAddress), func() {
		assertAvailableStaticPrecompilesRegistered(precompiles)
	})
}

type noopPrecompile struct{}

func (noopPrecompile) Address() common.Address { return common.Address{} }

func (noopPrecompile) RequiredGas([]byte) uint64 { return 0 }

func (noopPrecompile) Run(*vm.EVM, *vm.Contract, bool) ([]byte, error) {
	return nil, nil
}
