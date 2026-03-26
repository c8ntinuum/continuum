//go:build test
// +build test

package ibcbreaker

import (
	"testing"

	"github.com/cosmos/evm/evmd/tests/integration"
	"github.com/cosmos/evm/tests/integration/ibcbreaker"
)

func TestIbcBreakerWhitelistGovernance(t *testing.T) {
	ibcbreaker.IbcBreakerWhitelistGovernance(t, integration.CreateEvmd)
}
