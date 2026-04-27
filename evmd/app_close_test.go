package evmd

import (
	"testing"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/baseapp"
)

func TestEVMDCloseWithNilEVMMempool(t *testing.T) {
	app := &EVMD{
		BaseApp: baseapp.NewBaseApp("test", log.NewNopLogger(), dbm.NewMemDB(), nil),
	}

	require.NoError(t, app.Close())
}
