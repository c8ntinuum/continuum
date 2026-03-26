//go:build test
// +build test

package circuit

import (
	"testing"

	"github.com/cosmos/evm/evmd/tests/integration"
	"github.com/cosmos/evm/tests/integration/circuit"
)

func TestCircuitWhitelistGovernance(t *testing.T) {
	circuit.CircuitWhitelistGovernance(t, integration.CreateEvmd)
}
