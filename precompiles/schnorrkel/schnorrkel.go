package schnorrkel

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"

	schnorrkel "github.com/ChainSafe/go-schnorrkel"
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

const methodVerify = "verifySchnorrkelSignature"

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
	return common.HexToAddress(evmtypes.SchnorrkelPrecompileAddress)
}

func (p Precompile) RequiredGas(_ []byte) uint64 { return p.baseGas }

// verifySchnorrkelSignature(bytes signingCtx, bytes msg, bytes pubKey, bytes signature) -> (bool)
func (p *Precompile) Run(_ *vm.EVM, contract *vm.Contract, _ bool) ([]byte, error) {
	input := contract.Input
	if len(input) < 4 {
		return nil, vm.ErrExecutionReverted
	}
	m, err := p.MethodById(input[:4])
	if err != nil || m.Name != methodVerify {
		return nil, vm.ErrExecutionReverted
	}

	// Unpack to []any (compatible with older geth)
	vals, err := m.Inputs.Unpack(input[4:])
	if err != nil || len(vals) != 4 {
		return nil, vm.ErrExecutionReverted
	}

	signingCtx, ok := vals[0].([]byte)
	if !ok {
		out, _ := m.Outputs.Pack(false)
		return out, nil
	}
	msg, ok := vals[1].([]byte)
	if !ok {
		out, _ := m.Outputs.Pack(false)
		return out, nil
	}
	pubKeyBytes, ok := vals[2].([]byte)
	if !ok || len(pubKeyBytes) != 32 {
		out, _ := m.Outputs.Pack(false)
		return out, nil
	}
	sigBytes, ok := vals[3].([]byte)
	if !ok || len(sigBytes) != 64 {
		out, _ := m.Outputs.Pack(false)
		return out, nil
	}

	// Build transcript (same as your sample)
	transcript := schnorrkel.NewSigningContext(signingCtx, msg)

	// Parse public key: NewPublicKey([32]byte)
	var pkArr [32]byte
	copy(pkArr[:], pubKeyBytes)
	pubKey, err := schnorrkel.NewPublicKey(pkArr)
	if err != nil {
		out, _ := m.Outputs.Pack(false)
		return out, nil
	}

	// Parse signature: sig.Decode([64]byte)
	var sigArr [64]byte
	copy(sigArr[:], sigBytes)
	var sig schnorrkel.Signature
	if err := sig.Decode(sigArr); err != nil {
		out, _ := m.Outputs.Pack(false)
		return out, nil
	}

	// Verify (returns (bool, error))
	okv, err := pubKey.Verify(&sig, transcript)
	if err != nil {
		okv = false
	}

	out, packErr := m.Outputs.Pack(okv)
	if packErr != nil {
		return nil, errors.New("failed to ABI-pack result")
	}
	return out, nil
}
