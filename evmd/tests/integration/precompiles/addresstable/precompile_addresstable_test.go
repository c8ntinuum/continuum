package addresstable

import (
	"testing"

	"github.com/cosmos/evm/evmd/tests/integration"
	"github.com/cosmos/evm/tests/integration/precompiles/addresstable"
)

func TestAddressTablePrecompileIntegrationTestSuite(t *testing.T) {
	addresstable.TestAddressTablePrecompileIntegrationTestSuite(t, integration.CreateEvmd)
}
