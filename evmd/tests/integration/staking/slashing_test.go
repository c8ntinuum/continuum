package staking

import (
	"testing"

	"github.com/cosmos/evm/evmd/tests/integration"
	"github.com/cosmos/evm/tests/integration/staking"
)

func TestSlashingSigningInfoIntegration(t *testing.T) {
	staking.TestSlashingSigningInfoIntegration(t, integration.CreateEvmd)
}
