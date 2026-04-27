package keeper

import (
	"testing"

	"github.com/cosmos/evm/precompiles/reserved"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/require"
)

func TestWithStaticPrecompilesRejectsZeroAddress(t *testing.T) {
	precompile, err := reserved.NewPrecompile(15)
	require.NoError(t, err)

	keeper := &Keeper{}
	require.PanicsWithValue(t, "zero address cannot be registered as a static precompile", func() {
		keeper.WithStaticPrecompiles(map[common.Address]vm.PrecompiledContract{
			common.Address{}: precompile,
		})
	})
}

func TestGetStaticPrecompileInstanceReturnsErrorForMissingConfiguredPrecompile(t *testing.T) {
	address := common.HexToAddress(evmtypes.ReservedSlot48PrecompileAddress)
	keeper := &Keeper{
		precompiles: map[common.Address]vm.PrecompiledContract{},
	}
	params := &evmtypes.Params{
		ActiveStaticPrecompiles: []string{evmtypes.ReservedSlot48PrecompileAddress},
	}

	precompile, found, err := keeper.GetStaticPrecompileInstance(params, address)
	require.Nil(t, precompile)
	require.False(t, found)
	require.Error(t, err)
	require.ErrorIs(t, err, vm.ErrExecutionReverted)
	require.ErrorContains(t, err, evmtypes.ReservedSlot48PrecompileAddress)
}
