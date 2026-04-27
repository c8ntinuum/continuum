package pqslhdsa

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestRequiredGasUsesParamTier(t *testing.T) {
	precompile, err := NewPrecompile(250_000)
	require.NoError(t, err)

	method := ABI.Methods[MethodVerify]
	testCases := []struct {
		name    string
		paramID uint8
		want    uint64
	}{
		{name: "128f", paramID: uint8(SLH_SHA2_128F), want: slh128FGas},
		{name: "128s", paramID: uint8(SLH_SHA2_128S), want: slh128SGas},
		{name: "192f", paramID: uint8(SLH_SHA2_192F), want: slh192FGas},
		{name: "192s", paramID: uint8(SLH_SHA2_192S), want: slh192SGas},
		{name: "256f", paramID: uint8(SLH_SHA2_256F), want: slh256FGas},
		{name: "256s", paramID: uint8(SLH_SHA2_256S), want: slh256SGas},
		{name: "unknown", paramID: 99, want: slh256SGas},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args, err := method.Inputs.Pack(tc.paramID, [32]byte{}, []byte{}, []byte{})
			require.NoError(t, err)

			input := append(method.ID, args...)
			require.Equal(t, tc.want, precompile.RequiredGas(input))
		})
	}
}

func TestRunReturnsFalseForWrongSizedInputs(t *testing.T) {
	precompile, err := NewPrecompile(250_000)
	require.NoError(t, err)

	method := ABI.Methods[MethodVerify]
	args, err := method.Inputs.Pack(uint8(SLH_SHA2_128F), [32]byte{}, []byte{1}, []byte{2})
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
