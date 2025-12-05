package sp1verifiergroth16

import (
	"bytes"
	_ "embed"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	evmtypes "github.com/cosmos/evm/x/vm/types"
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

// Precompile defines the precompiled contract for sp1Verifier encoding.
type Precompile struct {
	abi.ABI
	baseGas uint64
}

// NewPrecompile creates a new sp1Verifier Precompile instance as a
// PrecompiledContract interface.
func NewPrecompile(baseGas uint64) (*Precompile, error) {
	if baseGas == 0 {
		return nil, fmt.Errorf("baseGas cannot be zero")
	}

	return &Precompile{
		ABI:     ABI,
		baseGas: baseGas,
	}, nil
}

// Address defines the address of the sp1Verifier precompiled contract.
func (Precompile) Address() common.Address {
	return common.HexToAddress(evmtypes.SP1VerifierGroth16PrecompileAddress)
}

// RequiredGas calculates the contract gas use.
func (p Precompile) RequiredGas(input []byte) uint64 {
	if len(input) < 4 {
		return p.baseGas
	}
	method, err := p.MethodById(input[:4])
	if err != nil {
		return p.baseGas
	}
	switch method.Name {
	case verifyProof:
		return p.baseGas
	case VERIFIER_HASH, VERSION:
		if p.baseGas < 10 {
			return 1
		}
		return p.baseGas / 10
	default:
		return p.baseGas
	}
}

// Run executes the precompiled contract sp1Verifier methods defined in the ABI.
func (p Precompile) Run(_ *vm.EVM, contract *vm.Contract, _ bool) (res []byte, err error) {
	// NOTE: This check avoid panicking when trying to decode the method ID
	if len(contract.Input) < 4 {
		return nil, vm.ErrExecutionReverted
	}
	methodID := contract.Input[:4]
	method, err := p.MethodById(methodID)
	if err != nil {
		return nil, err
	}
	argsBz := contract.Input[4:]
	args, err := method.Inputs.Unpack(argsBz)
	if err != nil {
		return nil, err
	}
	switch method.Name {
	case verifyProof:
		res, err = p.verifyProof(method, args)
	case VERIFIER_HASH:
		res, err = p.VERIFIER_HASH(method, args)
	case VERSION:
		res, err = p.VERSION(method, args)
	}
	if err != nil {
		return nil, err
	}
	return res, nil
}
