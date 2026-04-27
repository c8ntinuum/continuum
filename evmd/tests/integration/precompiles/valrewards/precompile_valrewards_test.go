package valrewards

import (
	"testing"

	"github.com/cosmos/evm/evmd/tests/integration"
	"github.com/cosmos/evm/tests/integration/precompiles/valrewards"
)

func TestValRewardsPrecompileIntegrationTestSuite(t *testing.T) {
	valrewards.TestPrecompileIntegrationTestSuite(t, integration.CreateEvmd)
}
