package ecvrf

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestRequiredGasScalesByWords(t *testing.T) {
	precompile, err := NewPrecompile(50_000)
	require.NoError(t, err)

	require.Equal(t, uint64(50_000), precompile.RequiredGas(nil))
	require.Equal(t, uint64(50_030), precompile.RequiredGas(make([]byte, 1)))
	require.Equal(t, uint64(50_060), precompile.RequiredGas(make([]byte, 33)))
}

func TestRunRejectsOversizedAlpha(t *testing.T) {
	precompile, err := NewPrecompile(50_000)
	require.NoError(t, err)

	method := ABI.Methods[methodVerify]
	args, err := method.Inputs.Pack("P256_SHA256_TAI", []byte{1}, make([]byte, maxECVRFAlphaBytes+1), []byte{2})
	require.NoError(t, err)

	contract := vm.NewContract(common.Address{}, common.Address{}, uint256.NewInt(0), 10_000_000, nil)
	contract.Input = append(method.ID, args...)

	_, err = precompile.Run(nil, contract, false)
	require.ErrorContains(t, err, "ecvrf alpha exceeds")
}
