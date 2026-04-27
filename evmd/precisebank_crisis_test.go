package evmd

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/evm/config"
	precisebankkeeper "github.com/cosmos/evm/x/precisebank/keeper"
	precisebanktypes "github.com/cosmos/evm/x/precisebank/types"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"cosmossdk.io/store/prefix"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestPreciseBankCrisisRoutesRegistered(t *testing.T) {
	app, _ := setup(false, 5, "precisebank-crisis-routes", config.EVMChainID)

	found := map[string]bool{}
	for _, route := range app.CrisisKeeper.Routes() {
		if route.ModuleName == precisebanktypes.ModuleName {
			found[fmt.Sprintf("%s/%s", route.ModuleName, route.Route)] = true
		}
	}

	require.Equal(t, map[string]bool{
		fmt.Sprintf("%s/%s", precisebanktypes.ModuleName, precisebankkeeper.NormalizedStateInvariantRoute): true,
		fmt.Sprintf("%s/%s", precisebanktypes.ModuleName, precisebankkeeper.ReserveBackingInvariantRoute):  true,
	}, found)
}

func TestPreciseBankCrisisAssertInvariantsOnCorruptState(t *testing.T) {
	app := Setup(t, "precisebank-crisis-check", config.EVMChainID)
	ctx := app.NewContextLegacy(false, tmproto.Header{
		ChainID: "precisebank-crisis-check",
		Height:  1,
	})

	store := prefix.NewStore(ctx.KVStore(app.GetKey(precisebanktypes.StoreKey)), precisebanktypes.FractionalBalancePrefix)
	invalid := precisebanktypes.ConversionFactor()
	bz, err := invalid.Marshal()
	require.NoError(t, err)
	store.Set(precisebanktypes.FractionalBalanceKey(sdk.AccAddress{0x42}), bz)

	panicValue := capturePanicValue(func() {
		app.CrisisKeeper.AssertInvariants(ctx)
	})
	require.NotNil(t, panicValue)
	require.Contains(t, fmt.Sprint(panicValue), precisebankkeeper.NormalizedStateInvariantRoute)
	require.Contains(t, fmt.Sprint(panicValue), precisebanktypes.ModuleName)
}

func capturePanicValue(fn func()) (panicValue any) {
	defer func() {
		panicValue = recover()
	}()

	fn()

	return nil
}
