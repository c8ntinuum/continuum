package frost

import (
	"testing"

	cmn "github.com/cosmos/evm/precompiles/common"
	frostlib "github.com/cosmos/evm/precompiles/frost/bytemare-stable/frost"
	frostdebug "github.com/cosmos/evm/precompiles/frost/bytemare-stable/frost/debug"
	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestRequiredGasUsesSignaturePricingForAllInputs(t *testing.T) {
	precompile, err := NewPrecompile(60_000)
	require.NoError(t, err)

	method := ABI.Methods[frostVerifySignature]
	args, err := method.Inputs.Pack(uint8(frostlib.Default), []byte("msg"), []byte("sig"), []byte("vk"))
	require.NoError(t, err)

	validInput := append(append([]byte{}, method.ID...), args...)
	unknownSelectorInput := append([]byte{0xde, 0xad, 0xbe, 0xef}, args...)

	require.Equal(t, cmn.LinearRequiredGas(precompile.baseGas, nil, frostVerifyPerWordGas), precompile.RequiredGas(nil))
	require.Equal(t, cmn.LinearRequiredGas(precompile.baseGas, validInput, frostVerifyPerWordGas), precompile.RequiredGas(validInput))
	require.Equal(t, cmn.LinearRequiredGas(precompile.baseGas, unknownSelectorInput, frostVerifyPerWordGas), precompile.RequiredGas(unknownSelectorInput))
}

func TestRunVerifiesValidSignature(t *testing.T) {
	precompile, err := NewPrecompile(60_000)
	require.NoError(t, err)

	ciphersuite := frostlib.Secp256k1
	group := ciphersuite.Group()
	message := []byte("continuum frost")
	secretKey := group.NewScalar().Random()
	verificationKey := group.Base().Multiply(secretKey)
	signature, err := frostdebug.Sign(ciphersuite, message, secretKey)
	require.NoError(t, err)

	out, err := runVerifyCall(
		t,
		precompile,
		ABI.Methods[frostVerifySignature],
		uint8(ciphersuite),
		message,
		signature.Encode(),
		verificationKey.Encode(),
	)
	require.NoError(t, err)
	require.True(t, unpackBoolOutput(t, ABI.Methods[frostVerifySignature], out))
}

func TestRunReturnsVerificationErrorForMismatchedKey(t *testing.T) {
	precompile, err := NewPrecompile(60_000)
	require.NoError(t, err)

	ciphersuite := frostlib.Secp256k1
	group := ciphersuite.Group()
	message := []byte("continuum frost")
	secretKey := group.NewScalar().Random()
	wrongSecretKey := group.NewScalar().Random()
	signature, err := frostdebug.Sign(ciphersuite, message, secretKey)
	require.NoError(t, err)

	_, err = runVerifyCall(
		t,
		precompile,
		ABI.Methods[frostVerifySignature],
		uint8(ciphersuite),
		message,
		signature.Encode(),
		group.Base().Multiply(wrongSecretKey).Encode(),
	)
	require.ErrorContains(t, err, "invalid Signature")
}

func TestRunRejectsOversizedMessage(t *testing.T) {
	precompile, err := NewPrecompile(60_000)
	require.NoError(t, err)

	method := ABI.Methods[frostVerifySignature]
	args, err := method.Inputs.Pack(uint8(frostlib.Default), make([]byte, maxFROSTMessageBytes+1), []byte("sig"), []byte("vk"))
	require.NoError(t, err)

	contract := newTestContract(append(append([]byte{}, method.ID...), args...))

	_, err = precompile.Run(nil, contract, false)
	require.ErrorContains(t, err, "frost message exceeds")
}

func TestABIExposesOnlySignatureVerification(t *testing.T) {
	_, found := ABI.Methods[frostVerifySignature]
	require.True(t, found)
	require.Len(t, ABI.Methods, 1)
}

func runVerifyCall(
	t *testing.T,
	precompile *Precompile,
	method gethabi.Method,
	ciphersuite uint8,
	message []byte,
	signature []byte,
	verificationKey []byte,
) ([]byte, error) {
	t.Helper()

	args, err := method.Inputs.Pack(ciphersuite, message, signature, verificationKey)
	require.NoError(t, err)

	contract := newTestContract(append(append([]byte{}, method.ID...), args...))
	return precompile.Run(nil, contract, false)
}

func unpackBoolOutput(t *testing.T, method gethabi.Method, output []byte) bool {
	t.Helper()

	values, err := method.Outputs.Unpack(output)
	require.NoError(t, err)
	require.Len(t, values, 1)

	ok, success := values[0].(bool)
	require.True(t, success)

	return ok
}

func newTestContract(input []byte) *vm.Contract {
	contract := vm.NewContract(common.Address{}, common.Address{}, uint256.NewInt(0), 10_000_000, nil)
	contract.Input = input

	return contract
}
