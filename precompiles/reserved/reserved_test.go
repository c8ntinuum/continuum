package reserved

import (
	"testing"

	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestReservedPrecompileAddresses(t *testing.T) {
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

	seen := make(map[common.Address]struct{}, len(expectedAddresses))
	for i, expectedAddress := range expectedAddresses {
		slot := minReservedSlot + i

		precompile, err := NewPrecompile(slot)
		require.NoError(t, err)

		address := precompile.Address()
		require.Equal(t, common.HexToAddress(expectedAddress), address)
		require.NotEqual(t, common.Address{}, address)

		_, exists := seen[address]
		require.Falsef(t, exists, "reserved slot %d reused address %s", slot, address)
		seen[address] = struct{}{}
	}
}

func TestNewPrecompileRejectsInvalidReservedSlots(t *testing.T) {
	_, err := NewPrecompile(minReservedSlot - 1)
	require.Error(t, err)

	_, err = NewPrecompile(maxReservedSlot + 1)
	require.Error(t, err)
}
