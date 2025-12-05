package sp1verifiergroth16

/*
#include <stdint.h>
#include <stdlib.h>

// Declare the Rust function
extern int verify_groth16_c(const uint8_t* proof_ptr, size_t proof_len,
                         const uint8_t* inputs_ptr, size_t inputs_len,
                         const char* hash_ptr);
*/
import "C"

import (
	"encoding/hex"
	"fmt"
	"unsafe"

	"github.com/ethereum/go-ethereum/accounts/abi"

	cmn "github.com/cosmos/evm/precompiles/common"
)

const (
	verifyProof   = "verifyProof"
	VERIFIER_HASH = "VERIFIER_HASH"
	VERSION       = "VERSION"
)

var verifierHash [32]byte

func init() {
	b, err := hex.DecodeString("a4594c59bbc142f3b81c3ecb7f50a7c34bc9af7c4c444b5d48b795427e285913")
	if err != nil {
		panic(err)
	}
	if len(b) != 32 {
		panic("VERIFIER_HASH must be 32 bytes")
	}
	copy(verifierHash[:], b)
}

func (p Precompile) verifyProof(
	_ *abi.Method,
	args []any,
) ([]byte, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 3, len(args))
	}
	programVKey, ok := args[0].([32]byte)
	if !ok {
		return nil, fmt.Errorf("invalid type for programVKey")
	}
	publicValues, ok := args[1].([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid type for publicValues")
	}
	proof, ok := args[2].([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid type for proofBytes")
	}

	if len(proof) == 0 || len(publicValues) == 0 {
		return nil, fmt.Errorf("proof and publicValues must be non-empty")
	}

	hashStr := C.CString("0x" + hex.EncodeToString(programVKey[:]))
	defer C.free(unsafe.Pointer(hashStr))

	res := C.verify_groth16_c(
		(*C.uint8_t)(unsafe.Pointer(&proof[0])), C.size_t(len(proof)),
		(*C.uint8_t)(unsafe.Pointer(&publicValues[0])), C.size_t(len(publicValues)),
		hashStr)

	switch res {
	case 0:
		return nil, nil
	case 1:
		return nil, fmt.Errorf("groth16 verifier: null or empty pointer (proof/publicInputs/hash)")
	case 2:
		return nil, fmt.Errorf("groth16 verifier: invalid UTF-8 in programVKey hash")
	case 3:
		return nil, fmt.Errorf("groth16 verifier: proof verification failed")
	case 4:
		return nil, fmt.Errorf("groth16 verifier: Groth16 vkey hash mismatch (wrong programVKey?)")
	case 5:
		return nil, fmt.Errorf("groth16 verifier: internal verifier error")
	default:
		return nil, fmt.Errorf("groth16 verifier: unknown error code %d", res)
	}
}

func (p Precompile) VERIFIER_HASH(method *abi.Method, args []interface{}) ([]byte, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 0, len(args))
	}
	return method.Outputs.Pack(verifierHash)
}

func (p Precompile) VERSION(
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 0, len(args))
	}
	return method.Outputs.Pack("SP1 - v5.0.0")
}
