package schnorrkel

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestRequiredGasScalesByWords(t *testing.T) {
	precompile, err := NewPrecompile(40_000)
	require.NoError(t, err)

	require.Equal(t, uint64(40_000), precompile.RequiredGas(nil))
	require.Equal(t, uint64(40_030), precompile.RequiredGas(make([]byte, 1)))
	require.Equal(t, uint64(40_060), precompile.RequiredGas(make([]byte, 33)))
}

func TestRunRejectsOversizedContextAndMessage(t *testing.T) {
	precompile, err := NewPrecompile(40_000)
	require.NoError(t, err)

	method := ABI.Methods[methodVerify]
	args, err := method.Inputs.Pack(
		make([]byte, maxSchnorrkelMessageBytes/2),
		make([]byte, (maxSchnorrkelMessageBytes/2)+1),
		make([]byte, 32),
		make([]byte, 64),
	)
	require.NoError(t, err)

	contract := vm.NewContract(common.Address{}, common.Address{}, uint256.NewInt(0), 10_000_000, nil)
	contract.Input = append(method.ID, args...)

	_, err = precompile.Run(nil, contract, false)
	require.ErrorContains(t, err, "schnorrkel signing context plus message exceeds")
}
