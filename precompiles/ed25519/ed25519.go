package ed25519

import (
	"crypto/ed25519"
	"embed"
	"errors"
	"fmt"

	cmn "github.com/cosmos/evm/precompiles/common"
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
	f   embed.FS
	ABI abi.ABI
)

func init() {
	var err error
	ABI, err = cmn.LoadABI(f, "abi.json")
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
// - 32 bytes of the ed25519 public key
// - 64 bytes of the solana ed25519 signature
// - 1952 bytes of the message signed (max 1952 chars UTF-8)
// Output data: ABI-encoded bool result and error
func (p *Precompile) Run(_ *vm.EVM, contract *vm.Contract, _ bool) (bz []byte, err error) {
	defer cmn.RecoverPrecompileError(&err)()

	input := contract.Input

	// NOTE: This check avoid panicking when trying to decode the method ID
	if len(input) < 4 {
		return nil, vm.ErrExecutionReverted
	}

	// NOTE: this function iterates over the method map and returns the method with the given ID
	methodID := input[:4]
	method, err := p.MethodById(methodID)
	if err != nil || method.Name != VerifyEd25519Signature {
		return nil, vm.ErrExecutionReverted
	}

	// Unpack the input data
	unpacked, err := method.Inputs.Unpack(input[4:])
	if err != nil || len(unpacked) != 3 {
		return nil, vm.ErrExecutionReverted
	}

	// Extract individual values
	pubKey := unpacked[0].([]byte)
	signature := unpacked[1].([]byte)
	message := unpacked[2].([]byte)

	if len(pubKey) != ed25519.PublicKeySize {
		return packBool(method, false)
	}
	if len(signature) != ed25519.SignatureSize {
		return packBool(method, false)
	}

	totalLength := len(pubKey) + len(signature) + len(message)
	if totalLength < MinInputLength || totalLength > MaxInputLength {
		return packBool(method, false)
	}

	return packBool(method, ed25519.Verify(pubKey, message, signature))
}

func packBool(method *abi.Method, value bool) ([]byte, error) {
	out, err := method.Outputs.Pack(value)
	if err != nil {
		return nil, errors.New("failed to ABI-pack result")
	}

	return out, nil
}
