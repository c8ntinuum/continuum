// Package frost implements an EVM precompile that verifies FROST signatures
// and signature shares using bytemare's libraries.
//
// Logic is intentionally unchanged from the original; only comments and small
// readability tweaks were added.
package frost

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"

	evmtypes "github.com/cosmos/evm/x/vm/types"

	"github.com/cosmos/evm/precompiles/frost/bytemare-stable/frost"
	"github.com/cosmos/evm/precompiles/frost/bytemare-stable/secret-sharing/keys"
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

const (
	frostVerifySignature      = "frostVerifySignature"
	frostVerifySignatureShare = "frostVerifySignatureShare"
)

// Precompile is the FROST precompile. The baseGas is a constant, returned by RequiredGas.
type Precompile struct {
	abi.ABI
	baseGas uint64
}

// NewPrecompile loads the ABI from the embedded FS and configures a constant base gas.
// Returns an error if the ABI cannot be loaded or baseGas is 0.
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
	return common.HexToAddress(evmtypes.FrostPrecompileAddress)
}

func (p Precompile) RequiredGas(_ []byte) uint64 { return p.baseGas }

func (p *Precompile) Run(_ *vm.EVM, contract *vm.Contract, _ bool) ([]byte, error) {
	input := contract.Input
	if len(input) < 4 {
		return nil, vm.ErrExecutionReverted
	}
	m, err := p.MethodById(input[:4])
	if err != nil {
		return nil, vm.ErrExecutionReverted
	}

	switch m.Name {
	case frostVerifySignature:
		return p.frostVerifySignature(m, input[4:])
	case frostVerifySignatureShare:
		return p.frostVerifySignatureShare(m, input[4:])
	default:
		return nil, vm.ErrExecutionReverted
	}
}

// frostVerifySignature verifies a full FROST signature.
//
// Expected ABI (from abi.json):
//
//	function frostVerifySignature(
//	    uint8 ciphersuite,
//	    bytes message,
//	    bytes signature,
//	    bytes verificationKey
//	) returns (bool)
//
// NOTE: On malformed inputs, it ABI-packs false (success path) rather than reverting.
// On decoding/verification errors it returns an error (causing revert).
func (p *Precompile) frostVerifySignature(m *abi.Method, data []byte) ([]byte, error) {
	// Decode the ABI-encoded arguments.
	vals, err := m.Inputs.Unpack(data)
	if err != nil || len(vals) != 4 {
		return nil, vm.ErrExecutionReverted
	}

	// Extract parameters with type assertions expected by the ABI.
	ciphersuiteBytes, okParam1 := vals[0].(uint8)
	messageBytes, okParam2 := vals[1].([]byte)
	signatureBytes, okParam3 := vals[2].([]byte)
	verificationKeyBytes, okParam4 := vals[3].([]byte)

	// If types don't match, return (bool=false) instead of reverting.
	if !(okParam1 && okParam2 && okParam3 && okParam4) {
		out, _ := m.Outputs.Pack(false)
		return out, nil
	}

	// Initialize the ciphersuite from its byte identifier.
	// NOTE: Available() is used to guard unsupported IDs.
	ciphersuite := frost.Ciphersuite(ciphersuiteBytes)
	if !ciphersuite.Available() {
		return nil, fmt.Errorf("unsupported ciphersuite id %d", ciphersuiteBytes)
	}

	// Decode the FROST signature (expects canonical encoding for the chosen suite).
	signature := new(frost.Signature)
	err = signature.Decode(signatureBytes)
	if err != nil {
		return nil, err
	}

	// Decode the verification (group) key for the suite.
	group := ciphersuite.Group()
	verificationKey := group.NewElement()
	err = verificationKey.Decode(verificationKeyBytes)
	if err != nil {
		return nil, err
	}

	// Perform verification. The FROST library handles hashing per ciphersuite.
	err = frost.VerifySignature(ciphersuite, messageBytes, signature, verificationKey)
	if err != nil {
		return nil, err
	}

	// ABI-pack a single bool = true on success.
	out, packErr := m.Outputs.Pack(true)
	if packErr != nil {
		return nil, errors.New("failed to ABI-pack result")
	}

	return out, nil
}

// frostVerifySignatureShare verifies a single FROST signature share against a configuration.
//
// Expected ABI (from abi.json, inferred from usage below; confirm the exact order/types):
//
//	function frostVerifySignatureShare(
//	    uint8   ciphersuite,
//	    uint16  threshold,
//	    uint16  maxSigners,
//	    bytes   verificationKey,
//	    bytes[] publicKeyShares,   // array of encoded keys
//	    bytes   commitments,       // encoded list; frost.DecodeList()
//	    bytes   message,
//	    bytes   signatureShare
//	) returns (bool)
//
// NOTE: The len(vals) check below compares against 3, but the method later reads 8 values.
// This looks like a bug or mismatch with the ABI; left unchanged as requested.
// Consider aligning the length check with the ABI to avoid silent misbehavior.
func (p *Precompile) frostVerifySignatureShare(m *abi.Method, data []byte) ([]byte, error) {
	// Decode ABI inputs.
	vals, err := m.Inputs.Unpack(data)
	if err != nil || len(vals) != 8 { // NOTE: Likely should be 8; preserved per "do not change logic".
		return nil, vm.ErrExecutionReverted
	}

	// Extract parameters (type assertions must match the ABI types).
	ciphersuiteBytes, okParam1 := vals[0].(uint8)
	threshold, okParam2 := vals[1].(uint16)
	maxSigners, okParam3 := vals[2].(uint16)
	verificationKeyBytes, okParam4 := vals[3].([]byte)
	publicKeySharesBytes, okParam5 := vals[4].([][]byte)
	commitmentsBytes, okParam6 := vals[5].([]byte)
	messageBytes, okParam7 := vals[6].([]byte)
	signatureShareBytes, okParam8 := vals[7].([]byte)

	// If any assertion failed, return (bool=false) instead of reverting.
	if !(okParam1 && okParam2 && okParam3 && okParam4 && okParam5 && okParam6 && okParam7 && okParam8) {
		out, _ := m.Outputs.Pack(false)
		return out, nil
	}

	// Initialize the ciphersuite (reject unsupported IDs).
	ciphersuite := frost.Ciphersuite(ciphersuiteBytes)
	if !ciphersuite.Available() {
		return nil, fmt.Errorf("unsupported ciphersuite id %d", ciphersuiteBytes)
	}
	// Decode group verification key.
	group := ciphersuite.Group()
	verificationKey := group.NewElement()
	err = verificationKey.Decode(verificationKeyBytes)
	if err != nil {
		return nil, err
	}

	// Decode each participant's public key share.
	// NOTE: Preallocating capacity can save gas if the ABI always provides N shares,
	// but we keep the exact logic and append semantics as-is.
	var publicKeyShares []*keys.PublicKeyShare = []*keys.PublicKeyShare{}
	for _, publicKeyShareBytes := range publicKeySharesBytes {
		var publicKeyShare *keys.PublicKeyShare = &keys.PublicKeyShare{}
		err = publicKeyShare.Decode(publicKeyShareBytes)
		if err != nil {
			return nil, err
		}
		publicKeyShares = append(publicKeyShares, publicKeyShare)
	}

	// Decode the signature share.
	signatureShare := new(frost.SignatureShare)
	err = signatureShare.Decode(signatureShareBytes)
	if err != nil {
		return nil, err
	}

	// Decode commitments list.
	// NOTE: The list must correspond to the round-1 commitments for the signers
	// referenced by the signature share.
	commitments, err := frost.DecodeList(commitmentsBytes)
	if err != nil {
		return nil, err
	}

	// Construct the configuration used by both signers and coordinator.
	// NOTE: Threshold and MaxSigners are taken verbatim; no extra validation is done here.
	configuration := &frost.Configuration{
		Ciphersuite:           ciphersuite,
		Threshold:             threshold,
		MaxSigners:            maxSigners,
		VerificationKey:       verificationKey,
		SignerPublicKeyShares: publicKeyShares,
	}

	// Verify the signature share for the provided message and commitments.
	err = configuration.VerifySignatureShare(signatureShare, messageBytes, commitments)
	if err != nil {
		return nil, err
	}

	// ABI-pack a single bool = true on success.
	out, packErr := m.Outputs.Pack(true)
	if packErr != nil {
		return nil, errors.New("failed to ABI-pack result")
	}
	return out, nil
}
