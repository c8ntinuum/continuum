// Package frost implements an EVM precompile that verifies caller-supplied
// FROST signatures using bytemare's libraries.
//
// It does not enforce any threshold group policy. Callers that need roster or
// threshold binding must enforce those checks at a higher layer.
package frost

import (
	"embed"
	"errors"
	"fmt"

	cmn "github.com/cosmos/evm/precompiles/common"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	"github.com/cosmos/evm/precompiles/frost/bytemare-stable/frost"
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
	frostVerifySignature  = "frostVerifySignature"
	frostVerifyPerWordGas = 30
	maxFROSTMessageBytes  = 64 * 1024
)

// Precompile is the FROST precompile. The baseGas is a constant, returned by RequiredGas.
type Precompile struct {
	abi.ABI
	baseGas uint64
}

// NewPrecompile loads the ABI from the embedded FS and configures a constant base gas.
// Returns an error if the ABI cannot be loaded or baseGas is 0.
func NewPrecompile(baseGas uint64) (*Precompile, error) {
	if baseGas == 0 {
		return nil, fmt.Errorf("baseGas cannot be zero")
	}

	return &Precompile{
		ABI:     ABI,
		baseGas: baseGas,
	}, nil
}

func (Precompile) Address() common.Address {
	return common.HexToAddress(evmtypes.FrostPrecompileAddress)
}

func (p Precompile) RequiredGas(input []byte) uint64 {
	return cmn.LinearRequiredGas(p.baseGas, input, frostVerifyPerWordGas)
}

func (p *Precompile) Run(_ *vm.EVM, contract *vm.Contract, _ bool) (bz []byte, err error) {
	defer cmn.RecoverPrecompileError(&err)()

	input := contract.Input
	if len(input) < 4 {
		return nil, vm.ErrExecutionReverted
	}
	m, err := p.MethodById(input[:4])
	if err != nil {
		return nil, vm.ErrExecutionReverted
	}

	switch m.Name {
	case frostVerifySignature:
		return p.frostVerifySignature(m, input[4:])
	default:
		return nil, vm.ErrExecutionReverted
	}
}

// frostVerifySignature verifies a full FROST signature.
//
// Expected ABI (from abi.json):
//
//	function frostVerifySignature(
//	    uint8 ciphersuite,
//	    bytes message,
//	    bytes signature,
//	    bytes verificationKey
//	) returns (bool)
//
// NOTE: On malformed inputs, it ABI-packs false (success path) rather than reverting.
// On decoding/verification errors it returns an error (causing revert).
func (p *Precompile) frostVerifySignature(m *abi.Method, data []byte) ([]byte, error) {
	// Decode the ABI-encoded arguments.
	vals, err := m.Inputs.Unpack(data)
	if err != nil || len(vals) != 4 {
		return nil, vm.ErrExecutionReverted
	}

	// Extract parameters with type assertions expected by the ABI.
	ciphersuiteBytes, okParam1 := vals[0].(uint8)
	messageBytes, okParam2 := vals[1].([]byte)
	signatureBytes, okParam3 := vals[2].([]byte)
	verificationKeyBytes, okParam4 := vals[3].([]byte)

	// If types don't match, return (bool=false) instead of reverting.
	if !(okParam1 && okParam2 && okParam3 && okParam4) {
		out, _ := m.Outputs.Pack(false)
		return out, nil
	}
	if len(messageBytes) > maxFROSTMessageBytes {
		return nil, fmt.Errorf("frost message exceeds %d bytes", maxFROSTMessageBytes)
	}

	// Initialize the ciphersuite from its byte identifier.
	// NOTE: Available() is used to guard unsupported IDs.
	ciphersuite := frost.Ciphersuite(ciphersuiteBytes)
	if !ciphersuite.Available() {
		return nil, fmt.Errorf("unsupported ciphersuite id %d", ciphersuiteBytes)
	}

	// Decode the FROST signature (expects canonical encoding for the chosen suite).
	signature := new(frost.Signature)
	err = signature.Decode(signatureBytes)
	if err != nil {
		return nil, err
	}

	// Decode the verification (group) key for the suite.
	group := ciphersuite.Group()
	verificationKey := group.NewElement()
	err = verificationKey.Decode(verificationKeyBytes)
	if err != nil {
		return nil, err
	}

	// Perform verification. The FROST library handles hashing per ciphersuite.
	err = frost.VerifySignature(ciphersuite, messageBytes, signature, verificationKey)
	if err != nil {
		return nil, err
	}

	// ABI-pack a single bool = true on success.
	out, packErr := m.Outputs.Pack(true)
	if packErr != nil {
		return nil, errors.New("failed to ABI-pack result")
	}

	return out, nil
}
