package schnorr

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"

	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/softwarecheng/btcd/btcec/v2/schnorr"
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
	MethodVerify = "verifySchnorrSignature"
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

// Address defines the address of the Schnorr precompiled contract.
func (Precompile) Address() common.Address {
	return common.HexToAddress(evmtypes.SchnorrPrecompileAddress)
}

// RequiredGas returns static gas (tune as needed).
func (p Precompile) RequiredGas(_ []byte) uint64 {
	return p.baseGas
}

// Run executes the precompile:
// verifySchnorrSignature(bytes32 xOnlyPubKey, bytes signature, bytes32 messageHash) -> (bool)
func (p *Precompile) Run(_ *vm.EVM, contract *vm.Contract, _ bool) ([]byte, error) {
	input := contract.Input
	if len(input) < 4 {
		return nil, vm.ErrExecutionReverted
	}

	method, err := p.MethodById(input[:4])
	if err != nil || method.Name != MethodVerify {
		return nil, vm.ErrExecutionReverted
	}

	// Unpack to []interface{} for compatibility with older go-ethereum
	values, err := method.Inputs.Unpack(input[4:])
	if err != nil || len(values) != 3 {
		return nil, vm.ErrExecutionReverted
	}

	// bytes32 -> [32]byte
	xOnly, ok := values[0].([32]byte)
	if !ok {
		out, _ := method.Outputs.Pack(false)
		return out, nil
	}

	// bytes -> []byte (must be exactly 64 bytes for BIP-340)
	sigBytes, ok := values[1].([]byte)
	if !ok || len(sigBytes) != 64 {
		out, _ := method.Outputs.Pack(false)
		return out, nil
	}

	// bytes32 -> [32]byte
	msgHash, ok := values[2].([32]byte)
	if !ok {
		out, _ := method.Outputs.Pack(false)
		return out, nil
	}

	// Parse/verify
	pub, err := schnorr.ParsePubKey(xOnly[:])
	if err != nil {
		out, _ := method.Outputs.Pack(false)
		return out, nil
	}
	sig, err := schnorr.ParseSignature(sigBytes)
	if err != nil {
		out, _ := method.Outputs.Pack(false)
		return out, nil
	}

	okv := sig.Verify(msgHash[:], pub)

	out, packErr := method.Outputs.Pack(okv)
	if packErr != nil {
		return nil, errors.New("failed to ABI-pack result")
	}
	return out, nil
}
