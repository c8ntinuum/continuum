package types

import (
	"testing"

	ratelimittypes "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	storetypes "cosmossdk.io/store/types"
	"github.com/stretchr/testify/require"
)

func TestStoreKeyDoesNotCollideWithIBCOrRateLimit(t *testing.T) {
	require.NotPanics(t, func() {
		storetypes.NewKVStoreKeys(ibcexported.StoreKey, ratelimittypes.StoreKey, StoreKey)
	})
}
