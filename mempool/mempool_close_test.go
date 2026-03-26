package mempool

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExperimentalEVMMempoolCloseNilReceiver(t *testing.T) {
	var mempool *ExperimentalEVMMempool

	require.NoError(t, mempool.Close())
}
