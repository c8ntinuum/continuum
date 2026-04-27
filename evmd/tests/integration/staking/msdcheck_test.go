package staking

import (
	"testing"

	"github.com/cosmos/evm/evmd/tests/integration"
	"github.com/cosmos/evm/tests/integration/staking"
)

func TestMSDCheckIntegration(t *testing.T) {
	staking.TestMSDCheckIntegration(t, integration.CreateEvmd)
}
