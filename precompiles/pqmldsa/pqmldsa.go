package pqmldsa

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"

	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	mldsa44 "github.com/trailofbits/ml-dsa/mldsa44"
	mldsa65 "github.com/trailofbits/ml-dsa/mldsa65"
	mldsa87 "github.com/trailofbits/ml-dsa/mldsa87"
	"github.com/trailofbits/ml-dsa/options"
)

var _ vm.PrecompiledContract = &Precompile{}

var (
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
	MethodVerify = "verify"
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

func (Precompile) Address() common.Address {
	return common.HexToAddress(evmtypes.PQMLDSAPrecompileAddress)
}

func (p Precompile) RequiredGas(_ []byte) uint64 {
	return p.baseGas
}

func (p *Precompile) Run(_ *vm.EVM, contract *vm.Contract, _ bool) ([]byte, error) {

	input := contract.Input
	if len(input) < 4 {
		return nil, vm.ErrExecutionReverted
	}

	method, err := p.MethodById(input[:4])
	if err != nil || method.Name != MethodVerify {
		return nil, vm.ErrExecutionReverted
	}

	values, err := method.Inputs.Unpack(input[4:])
	if err != nil || len(values) != 4 {
		return nil, vm.ErrExecutionReverted
	}

	// 0: uint8 scheme
	scheme, ok := values[0].(uint8)
	if !ok {
		return packBool(method, false)
	}

	// 1: bytes32 msgHash
	msgHash, ok := values[1].([32]byte)
	if !ok {
		return packBool(method, false)
	}

	// 2: bytes pubkey
	pubkeyBytes, ok := values[2].([]byte)
	if !ok {
		return packBool(method, false)
	}

	// 3: bytes signature
	sigBytes, ok := values[3].([]byte)
	if !ok {
		return packBool(method, false)
	}

	// Dispatch by scheme
	var valid bool
	switch scheme {
	case 44:
		valid = verify44(msgHash[:], pubkeyBytes, sigBytes)
	case 65:
		valid = verify65(msgHash[:], pubkeyBytes, sigBytes)
	case 87:
		valid = verify87(msgHash[:], pubkeyBytes, sigBytes)
	default:
		valid = false
	}

	return packBool(method, valid)
}

func packBool(method *abi.Method, v bool) ([]byte, error) {
	out, err := method.Outputs.Pack(v)
	if err != nil {
		return nil, errors.New("failed to ABI-pack result")
	}
	return out, nil
}

func verify44(msg []byte, pubkeyBytes, sigBytes []byte) bool {
	pub, err := mldsa44.PublicKeyFromBytes(pubkeyBytes)
	if err != nil || pub == nil {
		return false
	}
	return pub.VerifyWithOptions(msg, sigBytes, &options.Options{Context: "c8ntinuum-MLDSA-1"})
}

func verify65(msg []byte, pubkeyBytes, sigBytes []byte) bool {
	pub, err := mldsa65.PublicKeyFromBytes(pubkeyBytes)
	if err != nil || pub == nil {
		return false
	}
	return pub.VerifyWithOptions(msg, sigBytes, &options.Options{Context: "c8ntinuum-MLDSA-1"})
}

func verify87(msg []byte, pubkeyBytes, sigBytes []byte) bool {
	pub, err := mldsa87.PublicKeyFromBytes(pubkeyBytes)
	if err != nil || pub == nil {
		return false
	}
	return pub.VerifyWithOptions(msg, sigBytes, &options.Options{Context: "c8ntinuum-MLDSA-1"})
}
