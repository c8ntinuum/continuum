package pqmldsa

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestRequiredGasUsesSchemeTier(t *testing.T) {
	precompile, err := NewPrecompile(200_000)
	require.NoError(t, err)

	method := ABI.Methods[MethodVerify]
	testCases := []struct {
		name   string
		scheme uint8
		want   uint64
	}{
		{name: "44", scheme: 44, want: mldsa44Gas},
		{name: "65", scheme: 65, want: mldsa65Gas},
		{name: "87", scheme: 87, want: mldsa87Gas},
		{name: "unknown", scheme: 99, want: mldsa87Gas},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args, err := method.Inputs.Pack(tc.scheme, [32]byte{}, []byte{}, []byte{})
			require.NoError(t, err)

			input := append(method.ID, args...)
			require.Equal(t, tc.want, precompile.RequiredGas(input))
		})
	}
}

func TestRunReturnsFalseForWrongSizedInputs(t *testing.T) {
	precompile, err := NewPrecompile(200_000)
	require.NoError(t, err)

	method := ABI.Methods[MethodVerify]
	args, err := method.Inputs.Pack(uint8(44), [32]byte{}, []byte{1}, []byte{2})
	require.NoError(t, err)

	contract := vm.NewContract(common.Address{}, common.Address{}, uint256.NewInt(0), 10_000_000, nil)
	contract.Input = append(method.ID, args...)

	output, err := precompile.Run(nil, contract, false)
	require.NoError(t, err)

	decoded, err := method.Outputs.Unpack(output)
	require.NoError(t, err)
	require.Len(t, decoded, 1)
	require.False(t, decoded[0].(bool))
}
