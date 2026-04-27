package common

import "github.com/ethereum/go-ethereum/core/vm"

// CeilWords returns the number of 32-byte words needed to cover length bytes.
func CeilWords(length uint64) uint64 {
	if length == 0 {
		return 0
	}

	return (length + 31) / 32
}

// LinearRequiredGas prices calldata using a fixed base plus a per-32-byte-word charge.
func LinearRequiredGas(base uint64, input []byte, perWord uint64) uint64 {
	return LinearRequiredGasForLength(base, uint64(len(input)), perWord)
}

// LinearRequiredGasForLength prices a payload size using a fixed base plus a per-word charge.
func LinearRequiredGasForLength(base uint64, payloadLength uint64, perWord uint64) uint64 {
	return base + (CeilWords(payloadLength) * perWord)
}

// RecoverPrecompileError converts unexpected panics in stateless precompiles into EVM reverts.
func RecoverPrecompileError(err *error) func() {
	return func() {
		if recover() != nil {
			*err = vm.ErrExecutionReverted
		}
	}
}

// RecoverNativeActionError converts unexpected panics in stateful precompiles into EVM reverts.
func RecoverNativeActionError(err *error) func() {
	return RecoverPrecompileError(err)
}
