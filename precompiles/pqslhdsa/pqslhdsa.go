package pqslhdsa

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"

	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/trailofbits/go-slh-dsa/slh_dsa"
)

// Make sure we implement vm.PrecompiledContract
var _ vm.PrecompiledContract = &Precompile{}

//go:embed abi.json
var f []byte

var ABI abi.ABI

func init() {
	var err error
	ABI, err = abi.JSON(bytes.NewReader(f))
	if err != nil {
		panic(err)
	}
}

const (
	MethodVerify = "verify"

	slhContext = "c8ntinuum-SLHDSA-1"
)

// Solidity paramId mapping
type slhID uint8

const (
	SLH_SHA2_128F  slhID = 0
	SLH_SHA2_128S  slhID = 1
	SLH_SHA2_192F  slhID = 2
	SLH_SHA2_192S  slhID = 3
	SLH_SHA2_256F  slhID = 4
	SLH_SHA2_256S  slhID = 5
	SLH_SHAKE_128F slhID = 6
	SLH_SHAKE_128S slhID = 7
	SLH_SHAKE_192F slhID = 8
	SLH_SHAKE_192S slhID = 9
	SLH_SHAKE_256F slhID = 10
	SLH_SHAKE_256S slhID = 11
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
	return common.HexToAddress(evmtypes.PQSLHDSAPrecompileAddress)
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

	// 0: uint8 paramId
	paramId, ok := values[0].(uint8)
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

	// Dispatch by paramId
	var valid bool
	switch slhID(paramId) {
	case SLH_SHA2_128F:
		valid = verifySHA2_128F(msgHash[:], pubkeyBytes, sigBytes)
	case SLH_SHA2_128S:
		valid = verifySHA2_128S(msgHash[:], pubkeyBytes, sigBytes)
	case SLH_SHA2_192F:
		valid = verifySHA2_192F(msgHash[:], pubkeyBytes, sigBytes)
	case SLH_SHA2_192S:
		valid = verifySHA2_192S(msgHash[:], pubkeyBytes, sigBytes)
	case SLH_SHA2_256F:
		valid = verifySHA2_256F(msgHash[:], pubkeyBytes, sigBytes)
	case SLH_SHA2_256S:
		valid = verifySHA2_256S(msgHash[:], pubkeyBytes, sigBytes)
	case SLH_SHAKE_128F:
		valid = verifySHAKE_128F(msgHash[:], pubkeyBytes, sigBytes)
	case SLH_SHAKE_128S:
		valid = verifySHAKE_128S(msgHash[:], pubkeyBytes, sigBytes)
	case SLH_SHAKE_192F:
		valid = verifySHAKE_192F(msgHash[:], pubkeyBytes, sigBytes)
	case SLH_SHAKE_192S:
		valid = verifySHAKE_192S(msgHash[:], pubkeyBytes, sigBytes)
	case SLH_SHAKE_256F:
		valid = verifySHAKE_256F(msgHash[:], pubkeyBytes, sigBytes)
	case SLH_SHAKE_256S:
		valid = verifySHAKE_256S(msgHash[:], pubkeyBytes, sigBytes)
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

func verifySHA2_128F(msg []byte, pubkeyBytes, sigBytes []byte) bool {
	params := slh_dsa.SlhDsaSha2_128f()

	pk, err := slh_dsa.LoadPublicKey(params, pubkeyBytes)
	if err != nil {
		return false
	}

	sig, err := slh_dsa.LoadSignature(params, sigBytes)
	if err != nil {
		return false
	}

	return pk.Verify(sig, msg, []byte(slhContext))
}

func verifySHA2_128S(msg []byte, pubkeyBytes, sigBytes []byte) bool {
	params := slh_dsa.SlhDsaSha2_128s()

	pk, err := slh_dsa.LoadPublicKey(params, pubkeyBytes)
	if err != nil {
		return false
	}

	sig, err := slh_dsa.LoadSignature(params, sigBytes)
	if err != nil {
		return false
	}

	return pk.Verify(sig, msg, []byte(slhContext))
}

func verifySHA2_192F(msg []byte, pubkeyBytes, sigBytes []byte) bool {
	params := slh_dsa.SlhDsaSha2_192f()

	pk, err := slh_dsa.LoadPublicKey(params, pubkeyBytes)
	if err != nil {
		return false
	}

	sig, err := slh_dsa.LoadSignature(params, sigBytes)
	if err != nil {
		return false
	}

	return pk.Verify(sig, msg, []byte(slhContext))
}

func verifySHA2_192S(msg []byte, pubkeyBytes, sigBytes []byte) bool {
	params := slh_dsa.SlhDsaSha2_192s()

	pk, err := slh_dsa.LoadPublicKey(params, pubkeyBytes)
	if err != nil {
		return false
	}

	sig, err := slh_dsa.LoadSignature(params, sigBytes)
	if err != nil {
		return false
	}

	return pk.Verify(sig, msg, []byte(slhContext))
}

func verifySHA2_256F(msg []byte, pubkeyBytes, sigBytes []byte) bool {
	params := slh_dsa.SlhDsaSha2_256f()

	pk, err := slh_dsa.LoadPublicKey(params, pubkeyBytes)
	if err != nil {
		return false
	}

	sig, err := slh_dsa.LoadSignature(params, sigBytes)
	if err != nil {
		return false
	}

	return pk.Verify(sig, msg, []byte(slhContext))
}

func verifySHA2_256S(msg []byte, pubkeyBytes, sigBytes []byte) bool {
	params := slh_dsa.SlhDsaSha2_256s()

	pk, err := slh_dsa.LoadPublicKey(params, pubkeyBytes)
	if err != nil {
		return false
	}

	sig, err := slh_dsa.LoadSignature(params, sigBytes)
	if err != nil {
		return false
	}

	return pk.Verify(sig, msg, []byte(slhContext))
}

func verifySHAKE_128F(msg []byte, pubkeyBytes, sigBytes []byte) bool {
	params := slh_dsa.SlhDsaShake_128f()

	pk, err := slh_dsa.LoadPublicKey(params, pubkeyBytes)
	if err != nil {
		return false
	}

	sig, err := slh_dsa.LoadSignature(params, sigBytes)
	if err != nil {
		return false
	}

	return pk.Verify(sig, msg, []byte(slhContext))
}

func verifySHAKE_128S(msg []byte, pubkeyBytes, sigBytes []byte) bool {
	params := slh_dsa.SlhDsaShake_128s()

	pk, err := slh_dsa.LoadPublicKey(params, pubkeyBytes)
	if err != nil {
		return false
	}

	sig, err := slh_dsa.LoadSignature(params, sigBytes)
	if err != nil {
		return false
	}

	return pk.Verify(sig, msg, []byte(slhContext))
}

func verifySHAKE_192F(msg []byte, pubkeyBytes, sigBytes []byte) bool {
	params := slh_dsa.SlhDsaShake_192f()

	pk, err := slh_dsa.LoadPublicKey(params, pubkeyBytes)
	if err != nil {
		return false
	}

	sig, err := slh_dsa.LoadSignature(params, sigBytes)
	if err != nil {
		return false
	}

	return pk.Verify(sig, msg, []byte(slhContext))
}

func verifySHAKE_192S(msg []byte, pubkeyBytes, sigBytes []byte) bool {
	params := slh_dsa.SlhDsaShake_192s()

	pk, err := slh_dsa.LoadPublicKey(params, pubkeyBytes)
	if err != nil {
		return false
	}

	sig, err := slh_dsa.LoadSignature(params, sigBytes)
	if err != nil {
		return false
	}

	return pk.Verify(sig, msg, []byte(slhContext))
}

func verifySHAKE_256F(msg []byte, pubkeyBytes, sigBytes []byte) bool {
	params := slh_dsa.SlhDsaShake_256f()

	pk, err := slh_dsa.LoadPublicKey(params, pubkeyBytes)
	if err != nil {
		return false
	}

	sig, err := slh_dsa.LoadSignature(params, sigBytes)
	if err != nil {
		return false
	}

	return pk.Verify(sig, msg, []byte(slhContext))
}

func verifySHAKE_256S(msg []byte, pubkeyBytes, sigBytes []byte) bool {
	params := slh_dsa.SlhDsaShake_256s()

	pk, err := slh_dsa.LoadPublicKey(params, pubkeyBytes)
	if err != nil {
		return false
	}

	sig, err := slh_dsa.LoadSignature(params, sigBytes)
	if err != nil {
		return false
	}

	return pk.Verify(sig, msg, []byte(slhContext))
}
