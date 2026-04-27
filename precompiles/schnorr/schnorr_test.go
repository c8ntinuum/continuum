package schnorr

import (
	"crypto/sha256"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	btcschnorr "github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestRunReturnsTrueForValidSignature(t *testing.T) {
	precompile, err := NewPrecompile(15_000)
	require.NoError(t, err)

	privKey, err := btcec.NewPrivateKey()
	require.NoError(t, err)

	messageHash := sha256.Sum256([]byte("continuum schnorr"))
	signature, err := btcschnorr.Sign(privKey, messageHash[:])
	require.NoError(t, err)

	xOnlyPubKey := toBytes32(btcschnorr.SerializePubKey(privKey.PubKey()))

	out, err := runVerifyCall(t, precompile, ABI.Methods[MethodVerify], xOnlyPubKey, signature.Serialize(), messageHash)
	require.NoError(t, err)
	require.True(t, unpackBoolOutput(t, ABI.Methods[MethodVerify], out))
}

func TestRunReturnsFalseForInvalidSignature(t *testing.T) {
	precompile, err := NewPrecompile(15_000)
	require.NoError(t, err)

	privKey, err := btcec.NewPrivateKey()
	require.NoError(t, err)

	messageHash := sha256.Sum256([]byte("continuum schnorr"))
	signature, err := btcschnorr.Sign(privKey, messageHash[:])
	require.NoError(t, err)

	sigBytes := signature.Serialize()
	sigBytes[len(sigBytes)-1] ^= 0x01

	xOnlyPubKey := toBytes32(btcschnorr.SerializePubKey(privKey.PubKey()))

	out, err := runVerifyCall(t, precompile, ABI.Methods[MethodVerify], xOnlyPubKey, sigBytes, messageHash)
	require.NoError(t, err)
	require.False(t, unpackBoolOutput(t, ABI.Methods[MethodVerify], out))
}

func TestRunReturnsFalseForMalformedSignatureLength(t *testing.T) {
	precompile, err := NewPrecompile(15_000)
	require.NoError(t, err)

	messageHash := sha256.Sum256([]byte("continuum schnorr"))
	xOnlyPubKey := toBytes32(make([]byte, btcschnorr.PubKeyBytesLen))

	testCases := []struct {
		name      string
		signature []byte
	}{
		{name: "short signature", signature: make([]byte, btcschnorr.SignatureSize-1)},
		{name: "long signature", signature: make([]byte, btcschnorr.SignatureSize+1)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := runVerifyCall(t, precompile, ABI.Methods[MethodVerify], xOnlyPubKey, tc.signature, messageHash)
			require.NoError(t, err)
			require.False(t, unpackBoolOutput(t, ABI.Methods[MethodVerify], out))
		})
	}
}

func TestRunRevertsOnMalformedCalldata(t *testing.T) {
	precompile, err := NewPrecompile(15_000)
	require.NoError(t, err)

	method := ABI.Methods[MethodVerify]
	messageHash := sha256.Sum256([]byte("continuum schnorr"))
	args, err := method.Inputs.Pack(
		messageHash,
		make([]byte, btcschnorr.SignatureSize),
		messageHash,
	)
	require.NoError(t, err)

	testCases := []struct {
		name  string
		input []byte
	}{
		{name: "short calldata", input: []byte{0x01, 0x02, 0x03}},
		{name: "unknown selector", input: append([]byte{0xde, 0xad, 0xbe, 0xef}, args...)},
		{name: "truncated args", input: append([]byte{}, method.ID...)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			contract := newTestContract(tc.input)

			out, err := precompile.Run(nil, contract, false)
			require.Nil(t, out)
			require.ErrorIs(t, err, vm.ErrExecutionReverted)
		})
	}
}

func runVerifyCall(
	t *testing.T,
	precompile *Precompile,
	method abi.Method,
	xOnlyPubKey [32]byte,
	signature []byte,
	messageHash [32]byte,
) ([]byte, error) {
	t.Helper()

	args, err := method.Inputs.Pack(xOnlyPubKey, signature, messageHash)
	require.NoError(t, err)

	contract := newTestContract(append(append([]byte{}, method.ID...), args...))
	return precompile.Run(nil, contract, false)
}

func unpackBoolOutput(t *testing.T, method abi.Method, output []byte) bool {
	t.Helper()

	values, err := method.Outputs.Unpack(output)
	require.NoError(t, err)
	require.Len(t, values, 1)

	valid, ok := values[0].(bool)
	require.True(t, ok)

	return valid
}

func newTestContract(input []byte) *vm.Contract {
	contract := vm.NewContract(common.Address{}, common.Address{}, uint256.NewInt(0), 10_000_000, nil)
	contract.Input = input

	return contract
}

func toBytes32(b []byte) [32]byte {
	var out [32]byte
	copy(out[:], b)

	return out
}
