package sha3hash

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestRequiredGasScalesByWords(t *testing.T) {
	precompile, err := NewPrecompile(3_000)
	require.NoError(t, err)

	require.Equal(t, uint64(3_000), precompile.RequiredGas(nil))
	require.Equal(t, uint64(3_030), precompile.RequiredGas(make([]byte, 1)))
	require.Equal(t, uint64(3_060), precompile.RequiredGas(make([]byte, 33)))
}

func TestRunRejectsOversizedInput(t *testing.T) {
	precompile, err := NewPrecompile(3_000)
	require.NoError(t, err)

	method := ABI.Methods[methodHash]
	args, err := method.Inputs.Pack(make([]byte, maxSHA3Input+1), "SHA3-256")
	require.NoError(t, err)

	contract := vm.NewContract(common.Address{}, common.Address{}, uint256.NewInt(0), 10_000_000, nil)
	contract.Input = append(method.ID, args...)

	_, err = precompile.Run(nil, contract, false)
	require.ErrorContains(t, err, "sha3 input exceeds")
}
