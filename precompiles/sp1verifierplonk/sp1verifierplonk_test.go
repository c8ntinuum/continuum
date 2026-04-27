package sp1verifierplonk

import (
	"testing"

	cmn "github.com/cosmos/evm/precompiles/common"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestRequiredGasUsesVerifyPayloadSizes(t *testing.T) {
	precompile, err := NewPrecompile(800_000)
	require.NoError(t, err)

	method := ABI.Methods[verifyProof]
	publicValues := make([]byte, 64)
	proof := make([]byte, 96)
	args, err := method.Inputs.Pack([32]byte{}, publicValues, proof)
	require.NoError(t, err)

	input := append(method.ID, args...)
	expected := cmn.LinearRequiredGasForLength(precompile.baseGas, uint64(len(publicValues)+len(proof)), sp1VerifierPerWordGas)
	require.Equal(t, expected, precompile.RequiredGas(input))
}

func TestRequiredGasUsesCheapTierForMetadataMethods(t *testing.T) {
	precompile, err := NewPrecompile(800_000)
	require.NoError(t, err)

	method := ABI.Methods[VERIFIER_HASH]
	require.Equal(t, uint64(80_000), precompile.RequiredGas(method.ID))
}

func TestRunRejectsOversizedProof(t *testing.T) {
	precompile, err := NewPrecompile(800_000)
	require.NoError(t, err)

	method := ABI.Methods[verifyProof]
	args, err := method.Inputs.Pack([32]byte{}, []byte{1}, make([]byte, maxSP1ProofSize+1))
	require.NoError(t, err)

	contract := vm.NewContract(common.Address{}, common.Address{}, uint256.NewInt(0), 10_000_000, nil)
	contract.Input = append(method.ID, args...)

	_, err = precompile.Run(nil, contract, false)
	require.ErrorContains(t, err, "proof exceeds")
}

func TestRunRecoversVerifierPanics(t *testing.T) {
	precompile, err := NewPrecompile(800_000)
	require.NoError(t, err)

	original := verifyPlonkFn
	verifyPlonkFn = func([]byte, []byte, string) int {
		panic("boom")
	}
	defer func() {
		verifyPlonkFn = original
	}()

	method := ABI.Methods[verifyProof]
	args, err := method.Inputs.Pack([32]byte{}, []byte{1}, []byte{2})
	require.NoError(t, err)

	contract := vm.NewContract(common.Address{}, common.Address{}, uint256.NewInt(0), 10_000_000, nil)
	contract.Input = append(method.ID, args...)

	_, err = precompile.Run(nil, contract, false)
	require.ErrorIs(t, err, vm.ErrExecutionReverted)
}
