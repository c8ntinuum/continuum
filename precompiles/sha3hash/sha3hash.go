// sha3hash.go
package sha3hash

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"strings"

	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"golang.org/x/crypto/sha3"
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

const methodHash = "sha3Hash"

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
	return common.HexToAddress(evmtypes.Sha3HashPrecompileAddress)
}

// RequiredGas returns a constant gas (tune/extend with per-byte cost if desired).
func (p Precompile) RequiredGas(_ []byte) uint64 { return p.baseGas }

// Run executes: sha3Hash(bytes data, string hashName) -> (bytes)
func (p *Precompile) Run(_ *vm.EVM, contract *vm.Contract, _ bool) ([]byte, error) {
	input := contract.Input
	if len(input) < 4 {
		return nil, vm.ErrExecutionReverted
	}

	m, err := p.MethodById(input[:4])
	if err != nil || m.Name != methodHash {
		return nil, vm.ErrExecutionReverted
	}

	// Unpack to []any for broad ABI compatibility.
	vals, err := m.Inputs.Unpack(input[4:])
	if err != nil || len(vals) != 2 {
		return nil, vm.ErrExecutionReverted
	}

	data, ok := vals[0].([]byte)
	if !ok {
		out, _ := m.Outputs.Pack([]byte{})
		return out, nil
	}
	name, ok := vals[1].(string)
	if !ok {
		out, _ := m.Outputs.Pack([]byte{})
		return out, nil
	}

	algo := normalize(name)
	var digest []byte

	switch algo {
	case "SHA3-256", "SHA3_256":
		h := sha3.New256()
		_, _ = h.Write(data)
		digest = h.Sum(nil) // 32 bytes
	case "SHA3-512", "SHA3_512":
		h := sha3.New512()
		_, _ = h.Write(data)
		digest = h.Sum(nil) // 64 bytes
	default:
		// Unknown algorithm: return empty bytes (friendlier than revert).
		out, _ := m.Outputs.Pack([]byte{})
		return out, nil
	}

	out, packErr := m.Outputs.Pack(digest)
	if packErr != nil {
		return nil, errors.New("failed to ABI-pack result")
	}
	return out, nil
}

func normalize(s string) string {
	s = strings.TrimSpace(s)
	// standardize separators; accept diverse casings from callers
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ToUpper(s)
	return s
}
