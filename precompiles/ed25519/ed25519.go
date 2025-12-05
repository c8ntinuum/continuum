package ed25519

import (
	"bytes"
	"crypto/ed25519"
	_ "embed"
	"errors"
	"fmt"

	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

var _ vm.PrecompiledContract = &Precompile{}

var (
	// Embed abi json file to the executable binary. Needed when importing as dependency.
	//
	//go:embed abi.json
	f   []byte
	ABI abi.ABI
)

func init() {
	var err error
	ABI, err = abi.JSON(bytes.NewReader(f))
	if err != nil {
		panic(err)
	}
}

const (
	MaxInputLength         = 2048
	MinInputLength         = 97
	VerifyEd25519Signature = "verifyEd25519Signature"
)

type Precompile struct {
	abi.ABI
	baseGas uint64
}

func NewPrecompile(baseGas uint64) (*Precompile, error) {
	if baseGas == 0 {
		return nil, fmt.Errorf("baseGas cannot be zero")
	}

	return &Precompile{
		ABI:     ABI,
		baseGas: baseGas,
	}, nil
}

// Address defines the address of the ed25519 precompiled contract.
func (Precompile) Address() common.Address {
	return common.HexToAddress(evmtypes.Ed25519PrecompileAddress)
}

// RequiredGas returns the static gas required to execute the precompiled contract.
func (p Precompile) RequiredGas(_ []byte) uint64 {
	// return VerifyGas
	return p.baseGas
}

// Run executes the ed25519 signature verification
//
// Input data: max 2048 bytes of data including:
// - 32 bytes of the solana public key (address)
// - 64 bytes of the solana ed25519 signature
// - 1952 bytes of the message signed (max 1952 chars UTF-8)
// Output data: 1 byte of result data (0 false/1 true) and error
func (p *Precompile) Run(_ *vm.EVM, contract *vm.Contract, _ bool) ([]byte, error) {

	input := contract.Input

	// NOTE: This check avoid panicking when trying to decode the method ID
	if len(input) < 4 {
		return nil, vm.ErrExecutionReverted
	}

	// NOTE: this function iterates over the method map and returns the method with the given ID
	methodID := input[:4]
	method, err := p.MethodById(methodID)
	if err != nil {
		return nil, err
	}

	// Check the method name
	if method.Name != VerifyEd25519Signature {
		return nil, errors.New("invalid method name")
	}

	// Unpack the input data
	unpacked, err := method.Inputs.Unpack(input[4:])
	if err != nil {
		return nil, err
	}

	// Check the input length
	if len(unpacked[0].([]byte))+len(unpacked[1].([]byte))+len(unpacked[2].([]byte)) > MaxInputLength {
		// Input length is invalid
		return nil, errors.New("input length is invalid (too big, max 2048 bytes)")
	}
	// Check the input length
	if len(unpacked[0].([]byte))+len(unpacked[1].([]byte))+len(unpacked[2].([]byte)) < MinInputLength {
		// Input length is invalid
		return nil, errors.New("input length is invalid (too small, min 97 bytes)")
	}

	// Extract individual values
	addressBase58 := unpacked[0].([]byte)
	signatureBase64 := unpacked[1].([]byte)
	message := unpacked[2].([]byte)
	// Verify the signature
	isValid := ed25519.Verify(addressBase58, message, signatureBase64)
	if !isValid {
		return []byte{0}, nil
	}
	return []byte{1}, nil
}
