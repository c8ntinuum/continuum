package ed25519

import (
	stded25519 "crypto/ed25519"
	"crypto/rand"
	"math/big"
	"strings"
	"testing"

	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestRunReturnsABIPackedTrueForValidSignature(t *testing.T) {
	precompile, err := NewPrecompile(12_000)
	require.NoError(t, err)

	pubKey, privKey, err := stded25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	message := []byte("continuum")
	signature := stded25519.Sign(privKey, message)

	out, err := runVerifyCall(t, precompile, ABI.Methods[VerifyEd25519Signature], pubKey, signature, message)
	require.NoError(t, err)
	require.Len(t, out, 32)
	require.True(t, unpackBoolOutput(t, ABI.Methods[VerifyEd25519Signature], out))
}

func TestRunReturnsABIPackedFalseForInvalidSignature(t *testing.T) {
	precompile, err := NewPrecompile(12_000)
	require.NoError(t, err)

	pubKey, privKey, err := stded25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	message := []byte("continuum")
	signature := stded25519.Sign(privKey, message)
	signature[len(signature)-1] ^= 0x01

	out, err := runVerifyCall(t, precompile, ABI.Methods[VerifyEd25519Signature], pubKey, signature, message)
	require.NoError(t, err)
	require.Len(t, out, 32)
	require.False(t, unpackBoolOutput(t, ABI.Methods[VerifyEd25519Signature], out))
}

func TestRunReturnsABIPackedFalseForMalformedDecodedInputs(t *testing.T) {
	precompile, err := NewPrecompile(12_000)
	require.NoError(t, err)

	messageLimit := MaxInputLength - stded25519.PublicKeySize - stded25519.SignatureSize
	testCases := []struct {
		name      string
		pubKey    []byte
		signature []byte
		message   []byte
	}{
		{
			name:      "short pubkey",
			pubKey:    make([]byte, stded25519.PublicKeySize-1),
			signature: make([]byte, stded25519.SignatureSize),
			message:   []byte("a"),
		},
		{
			name:      "long pubkey",
			pubKey:    make([]byte, stded25519.PublicKeySize+1),
			signature: make([]byte, stded25519.SignatureSize),
			message:   []byte("a"),
		},
		{
			name:      "short signature",
			pubKey:    make([]byte, stded25519.PublicKeySize),
			signature: make([]byte, stded25519.SignatureSize-1),
			message:   []byte("a"),
		},
		{
			name:      "long signature",
			pubKey:    make([]byte, stded25519.PublicKeySize),
			signature: make([]byte, stded25519.SignatureSize+1),
			message:   []byte("a"),
		},
		{
			name:      "empty message",
			pubKey:    make([]byte, stded25519.PublicKeySize),
			signature: make([]byte, stded25519.SignatureSize),
			message:   []byte{},
		},
		{
			name:      "oversized payload",
			pubKey:    make([]byte, stded25519.PublicKeySize),
			signature: make([]byte, stded25519.SignatureSize),
			message:   make([]byte, messageLimit+1),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := runVerifyCall(t, precompile, ABI.Methods[VerifyEd25519Signature], tc.pubKey, tc.signature, tc.message)
			require.NoError(t, err)
			require.Len(t, out, 32)
			require.False(t, unpackBoolOutput(t, ABI.Methods[VerifyEd25519Signature], out))
		})
	}
}

func TestRunRevertsOnStructuralMalformedCalldata(t *testing.T) {
	precompile, err := NewPrecompile(12_000)
	require.NoError(t, err)

	method := ABI.Methods[VerifyEd25519Signature]
	args, err := method.Inputs.Pack(
		make([]byte, stded25519.PublicKeySize),
		make([]byte, stded25519.SignatureSize),
		[]byte("a"),
	)
	require.NoError(t, err)

	testCases := []struct {
		name  string
		input []byte
	}{
		{
			name:  "short calldata",
			input: []byte{0x01, 0x02, 0x03},
		},
		{
			name:  "unknown selector",
			input: append([]byte{0xde, 0xad, 0xbe, 0xef}, args...),
		},
		{
			name:  "truncated abi payload",
			input: append([]byte{}, method.ID...),
		},
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

func TestRunRecoversUnexpectedPanics(t *testing.T) {
	badABI, err := gethabi.JSON(strings.NewReader(`[
		{
			"type":"function",
			"name":"verifyEd25519Signature",
			"stateMutability":"pure",
			"inputs":[
				{"name":"pubKey","type":"uint256"},
				{"name":"signature","type":"bytes"},
				{"name":"message","type":"bytes"}
			],
			"outputs":[
				{"name":"success","type":"bool"}
			]
		}
	]`))
	require.NoError(t, err)

	precompile := &Precompile{
		ABI:     badABI,
		baseGas: 12_000,
	}

	method := badABI.Methods[VerifyEd25519Signature]
	args, err := method.Inputs.Pack(big.NewInt(1), make([]byte, stded25519.SignatureSize), []byte("a"))
	require.NoError(t, err)

	contract := newTestContract(append(append([]byte{}, method.ID...), args...))

	out, err := precompile.Run(nil, contract, false)
	require.Nil(t, out)
	require.ErrorIs(t, err, vm.ErrExecutionReverted)
}

func runVerifyCall(
	t *testing.T,
	precompile *Precompile,
	method gethabi.Method,
	pubKey []byte,
	signature []byte,
	message []byte,
) ([]byte, error) {
	t.Helper()

	args, err := method.Inputs.Pack(pubKey, signature, message)
	require.NoError(t, err)

	contract := newTestContract(append(append([]byte{}, method.ID...), args...))
	return precompile.Run(nil, contract, false)
}

func unpackBoolOutput(t *testing.T, method gethabi.Method, output []byte) bool {
	t.Helper()

	values, err := method.Outputs.Unpack(output)
	require.NoError(t, err)
	require.Len(t, values, 1)

	success, ok := values[0].(bool)
	require.True(t, ok)

	return success
}

func newTestContract(input []byte) *vm.Contract {
	contract := vm.NewContract(common.Address{}, common.Address{}, uint256.NewInt(0), 10_000_000, nil)
	contract.Input = input

	return contract
}
