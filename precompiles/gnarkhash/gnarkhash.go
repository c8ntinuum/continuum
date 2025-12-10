// gnarkhash.go
package gnarkhash

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	_ "embed"

	evmtypes "github.com/cosmos/evm/x/vm/types"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	ghash "github.com/consensys/gnark-crypto/hash"
	_ "github.com/consensys/gnark-crypto/hash/all"
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

const methodHash = "gnarkHash"

type Precompile struct {
	abi.ABI
	baseGas uint64
}

// NewPrecompile constructs the precompile with a fixed base gas.
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
	return common.HexToAddress(evmtypes.GnarkHashPrecompileAddress)
}

func (p Precompile) RequiredGas(_ []byte) uint64 { return p.baseGas }

// Run executes: gnarkHash(bytes data, string hashName) -> (bytes)
func (p *Precompile) Run(_ *vm.EVM, contract *vm.Contract, _ bool) ([]byte, error) {
	input := contract.Input
	if len(input) < 4 {
		return nil, vm.ErrExecutionReverted
	}

	m, err := p.MethodById(input[:4])
	if err != nil || m.Name != methodHash {
		return nil, vm.ErrExecutionReverted
	}

	// Unpack args into []any for broad geth-ABI compatibility.
	vals, err := m.Inputs.Unpack(input[4:])
	if err != nil || len(vals) != 2 {
		return nil, vm.ErrExecutionReverted
	}

	data, ok := vals[0].([]byte)
	if !ok {
		// Bad ABI: return empty digest (not reverting keeps UX friendlier).
		out, _ := m.Outputs.Pack([]byte{})
		return out, nil
	}
	hashName, ok := vals[1].(string)
	if !ok {
		out, _ := m.Outputs.Pack([]byte{})
		return out, nil
	}

	// Normalize the hash name (gnark expects exact constants; make it lenient for callers).
	name := strings.ToUpper(strings.TrimSpace(hashName))
	// Instantiate hasher (panic-safe; NewHash panics if name is unknown/unregistered).
	h, newErr := newHashSafe(name)
	if newErr != nil {
		out, _ := m.Outputs.Pack([]byte{})
		return out, nil
	}
	// Hash the data.
	_, _ = h.Write(data)                   // hash.Hash never returns non-nil error
	digest := h.Sum(nil)                   // length depends on hash (e.g., 32/48/96/4/8)
	out, packErr := m.Outputs.Pack(digest) // ABI-encode bytes
	if packErr != nil {
		return nil, errors.New("failed to ABI-pack result")
	}
	return out, nil
}

// newHashSafe wraps ghash.NewHash(name) which panics on unknown names.
func newHashSafe(name string) (h interface {
	Write([]byte) (int, error)
	Sum([]byte) []byte
}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("hash %q not available: %v", name, r)
		}
	}()
	return ghash.NewHash(name), nil
}
