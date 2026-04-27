package integration

import (
	"testing"

	"github.com/cosmos/evm/tests/integration/x/feeburn"
)

func TestFeeCollectorBurnBeforeDistribution(t *testing.T) {
	feeburn.TestBurnFeeCollectorBeforeDistribution(t, CreateEvmd)
}
