package json

import (
	"bytes"
	_ "embed"
	gjson "encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

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

type Precompile struct {
	abi.ABI
	baseGas uint64
}

const (
	extractAsBytes          = "extractAsBytes"
	extractAsBytesList      = "extractAsBytesList"
	extractAsUint256        = "extractAsUint256"
	extractAsBytesFromArray = "extractAsBytesFromArray"
)

// NewPrecompile creates a new json Precompile instance as a PrecompiledContract interface.
func NewPrecompile(baseGas uint64) (*Precompile, error) {
	if baseGas == 0 {
		return nil, fmt.Errorf("baseGas cannot be zero")
	}

	return &Precompile{
		ABI:     ABI,
		baseGas: baseGas,
	}, nil
}

// Address defines the address of the JSON precompiled contract.
func (Precompile) Address() common.Address {
	return common.HexToAddress(evmtypes.JsonPrecompileAddress)
}

// RequiredGas returns the static gas required to execute the precompiled contract.
func (p Precompile) RequiredGas(input []byte) uint64 {
	return p.baseGas + uint64(len(input))
}

func (p *Precompile) Run(_ *vm.EVM, contract *vm.Contract, _ bool) (res []byte, err error) {
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
	case extractAsBytes:
		res, err = p.extractAsBytes(method, args)
	case extractAsBytesList:
		res, err = p.extractAsBytesList(method, args)
	case extractAsUint256:
		byteArr := make([]byte, 32)
		bi, err := p.extractAsUint256(args)
		if err != nil {
			return nil, err
		}
		if bi.BitLen() > 256 {
			return nil, errors.New("value does not fit in 32 bytes")
		}

		bi.FillBytes(byteArr)
		return byteArr, nil
	case extractAsBytesFromArray:
		res, err = p.extractAsBytesFromArray(method, args)
	}
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (p Precompile) extractAsBytes(
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 2, len(args))
	}
	bz := args[0].([]byte)
	decoded := map[string]gjson.RawMessage{}
	if err := gjson.Unmarshal(bz, &decoded); err != nil {
		return nil, err
	}
	key := args[1].(string)
	result, ok := decoded[key]
	if !ok {
		return nil, fmt.Errorf("input does not contain key %s", key)
	}
	// in the case of a string value, remove the quotes
	if len(result) >= 2 && result[0] == '"' && result[len(result)-1] == '"' {
		result = result[1 : len(result)-1]
	}

	return method.Outputs.Pack([]byte(result))
}

func (p Precompile) extractAsBytesList(
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 2, len(args))
	}
	bz := args[0].([]byte)
	decoded := map[string]gjson.RawMessage{}
	if err := gjson.Unmarshal(bz, &decoded); err != nil {
		return nil, err
	}
	key := args[1].(string)
	result, ok := decoded[key]
	if !ok {
		return nil, fmt.Errorf("input does not contain key %s", key)
	}
	decodedResult := []gjson.RawMessage{}
	if err := gjson.Unmarshal(result, &decodedResult); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(Map(decodedResult, func(r gjson.RawMessage) []byte { return []byte(r) }))
}

func (p Precompile) extractAsUint256(
	args []interface{},
) (*big.Int, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 2, len(args))
	}
	bz := args[0].([]byte)
	decoded := map[string]gjson.RawMessage{}
	if err := gjson.Unmarshal(bz, &decoded); err != nil {
		return nil, err
	}
	key := args[1].(string)
	result, ok := decoded[key]
	if !ok {
		return nil, fmt.Errorf("input does not contain key %s", key)
	}

	// Assuming result is your byte slice
	// Convert byte slice to string and trim quotation marks
	strValue := strings.Trim(string(result), "\"")

	// Convert the string to big.Int
	value, success := new(big.Int).SetString(strValue, 10)
	if !success {
		return nil, fmt.Errorf("failed to convert %s to big.Int", strValue)
	}

	return value, nil
}

func (p Precompile) extractAsBytesFromArray(
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 2, len(args))
	}
	bz := args[0].([]byte)
	var decoded []gjson.RawMessage
	if err := gjson.Unmarshal(bz, &decoded); err != nil {
		return nil, err
	}
	if len(decoded) > 1<<16 {
		return nil, errors.New("input array is larger than 2^16")
	}
	index, ok := args[1].(uint16)
	if !ok {
		return nil, errors.New("index must be uint16")
	}
	if int(index) >= len(decoded) {
		return nil, fmt.Errorf("index %d is out of bounds", index)
	}
	result := decoded[index]

	// in the case of a string value, remove the quotes
	if len(result) >= 2 && result[0] == '"' && result[len(result)-1] == '"' {
		result = result[1 : len(result)-1]
	}

	return method.Outputs.Pack([]byte(result))
}

func Map[I any, O any](input []I, lambda func(i I) O) []O {
	res := []O{}
	for _, i := range input {
		res = append(res, lambda(i))
	}
	return res
}
