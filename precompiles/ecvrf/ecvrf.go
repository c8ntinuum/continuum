// ecvrf.go
package ecvrf

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	_ "embed"
	"errors"
	"fmt"
	"strings"

	evmtypes "github.com/cosmos/evm/x/vm/types"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"

	ecvrf "github.com/vechain/go-ecvrf"
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

const methodVerify = "ecvrfVerify"

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
	return common.HexToAddress(evmtypes.EcvrfPrecompileAddress)
}

func (p Precompile) RequiredGas(_ []byte) uint64 { return p.baseGas }

// ecvrfVerify(string suite, bytes pubKey, bytes alpha, bytes pi) -> (bool ok, bytes beta)
func (p *Precompile) Run(_ *vm.EVM, contract *vm.Contract, _ bool) ([]byte, error) {
	input := contract.Input
	if len(input) < 4 {
		return nil, vm.ErrExecutionReverted
	}
	m, err := p.MethodById(input[:4])
	if err != nil || m.Name != methodVerify {
		return nil, vm.ErrExecutionReverted
	}

	vals, err := m.Inputs.Unpack(input[4:])
	if err != nil || len(vals) != 4 {
		return nil, vm.ErrExecutionReverted
	}

	suite, ok := vals[0].(string)
	if !ok {
		out, _ := m.Outputs.Pack(false, []byte{})
		return out, nil
	}
	pubKeyBytes, ok := vals[1].([]byte)
	if !ok {
		out, _ := m.Outputs.Pack(false, []byte{})
		return out, nil
	}
	alpha, ok := vals[2].([]byte)
	if !ok {
		out, _ := m.Outputs.Pack(false, []byte{})
		return out, nil
	}
	pi, ok := vals[3].([]byte)
	if !ok {
		out, _ := m.Outputs.Pack(false, []byte{})
		return out, nil
	}

	s := normalizeSuite(suite)

	var (
		pk   *ecdsa.PublicKey
		beta []byte
		vErr error
	)

	switch s {
	case "SECP256K1_SHA256_TAI":
		pk, vErr = parseSecp256k1Pub(pubKeyBytes)
		if vErr == nil {
			beta, vErr = ecvrf.Secp256k1Sha256Tai.Verify(pk, alpha, pi)
		}
	case "P256_SHA256_TAI":
		pk, vErr = parseP256Pub(pubKeyBytes)
		if vErr == nil {
			beta, vErr = ecvrf.P256Sha256Tai.Verify(pk, alpha, pi)
		}
	default:
		out, _ := m.Outputs.Pack(false, []byte{})
		return out, nil
	}

	okv := vErr == nil
	if !okv {
		beta = []byte{}
	}
	out, packErr := m.Outputs.Pack(okv, beta)
	if packErr != nil {
		return nil, errors.New("failed to ABI-pack result")
	}
	return out, nil
}

func normalizeSuite(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "-", "_")
	return strings.ToUpper(s)
}

func parseSecp256k1Pub(b []byte) (*ecdsa.PublicKey, error) {
	switch len(b) {
	case 33: // compressed SEC1
		return gethcrypto.DecompressPubkey(b)
	case 65: // uncompressed SEC1
		return gethcrypto.UnmarshalPubkey(b)
	default:
		return nil, fmt.Errorf("bad secp256k1 pubkey length: %d", len(b))
	}
}

func parseP256Pub(b []byte) (*ecdsa.PublicKey, error) {
	c := elliptic.P256()
	switch len(b) {
	case 65: // uncompressed
		x, y := elliptic.Unmarshal(c, b)
		if x == nil || y == nil {
			return nil, errors.New("invalid p256 uncompressed pubkey")
		}
		return &ecdsa.PublicKey{Curve: c, X: x, Y: y}, nil
	case 33: // compressed (Go 1.20+: elliptic.UnmarshalCompressed)
		x, y := elliptic.UnmarshalCompressed(c, b)
		if x == nil || y == nil {
			return nil, errors.New("invalid p256 compressed pubkey")
		}
		return &ecdsa.PublicKey{Curve: c, X: x, Y: y}, nil
	default:
		return nil, fmt.Errorf("bad p256 pubkey length: %d", len(b))
	}
}
