package utils

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

// ParseDelegation defensively recognizes EIP-7702 delegation code before
// delegating to the upstream helper. This keeps call sites stable even if the
// dependency behavior changes on malformed inputs.
func ParseDelegation(code []byte) (common.Address, bool) {
	if len(code) != len(ethtypes.DelegationPrefix)+common.AddressLength {
		return common.Address{}, false
	}
	if !bytes.HasPrefix(code, ethtypes.DelegationPrefix) {
		return common.Address{}, false
	}

	return ethtypes.ParseDelegation(code)
}
