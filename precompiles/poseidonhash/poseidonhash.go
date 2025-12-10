package poseidonhash

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"math/big"

	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	poseidonv2 "github.com/iden3/go-iden3-crypto/v2/poseidon"
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
	MethodPoseidon = "poseidonHash"
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
	return common.HexToAddress(evmtypes.PoseidonHashPrecompileAddress)
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
	if err != nil || method.Name != MethodPoseidon {
		return nil, vm.ErrExecutionReverted
	}

	values, err := method.Inputs.Unpack(input[4:])
	if err != nil || len(values) != 1 {
		return nil, vm.ErrExecutionReverted
	}

	var a, b, c *big.Int

	if arr, ok := values[0].([3]*big.Int); ok {
		a = arr[0]
		b = arr[1]
		c = arr[2]
	} else if slice, ok := values[0].([]*big.Int); ok && len(slice) == 3 {
		a = slice[0]
		b = slice[1]
		c = slice[2]
	} else {
		return nil, vm.ErrExecutionReverted
	}

	if a == nil || b == nil || c == nil {
		return nil, vm.ErrExecutionReverted
	}
	hash, err := poseidonv2.Hash([]*big.Int{a, b, c})
	if err != nil {
		return nil, vm.ErrExecutionReverted
	}

	out, packErr := method.Outputs.Pack(hash)
	if packErr != nil {
		return nil, errors.New("failed to ABI-pack result")
	}

	return out, nil
}
