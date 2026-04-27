package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/evm/x/precisebank/keeper"
	"github.com/cosmos/evm/x/precisebank/types"

	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store/prefix"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type invariantRoute struct {
	module string
	route  string
}

type invariantRegistryStub struct {
	routes []invariantRoute
}

func (s *invariantRegistryStub) RegisterRoute(moduleName, route string, _ sdk.Invariant) {
	s.routes = append(s.routes, invariantRoute{module: moduleName, route: route})
}

func TestRegisterInvariants(t *testing.T) {
	td := newMockedTestData(t)
	registry := &invariantRegistryStub{}

	keeper.RegisterInvariants(registry, &td.keeper)

	require.ElementsMatch(t, []invariantRoute{
		{module: types.ModuleName, route: keeper.NormalizedStateInvariantRoute},
		{module: types.ModuleName, route: keeper.ReserveBackingInvariantRoute},
	}, registry.routes)
}

func TestNormalizedStateInvariant(t *testing.T) {
	t.Run("valid state passes", func(t *testing.T) {
		td := newMockedTestData(t)
		td.ak.EXPECT().GetModuleAddress(types.ModuleName).Return(sdk.AccAddress{100}).Once()

		td.keeper.SetFractionalBalance(td.ctx, sdk.AccAddress{1}, sdkmath.NewInt(123))
		td.keeper.SetRemainderAmount(td.ctx, sdkmath.NewInt(456))

		msg, broken := keeper.NormalizedStateInvariant(&td.keeper)(td.ctx)
		require.False(t, broken)
		require.Empty(t, msg)
	})

	t.Run("invalid fractional balance row breaks invariant", func(t *testing.T) {
		td := newMockedTestData(t)
		td.ak.EXPECT().GetModuleAddress(types.ModuleName).Return(sdk.AccAddress{100}).Once()

		store := prefix.NewStore(td.ctx.KVStore(td.storeKey), types.FractionalBalancePrefix)
		invalid := types.ConversionFactor()
		bz, err := invalid.Marshal()
		require.NoError(t, err)
		store.Set(types.FractionalBalanceKey(sdk.AccAddress{1}), bz)

		msg, broken := keeper.NormalizedStateInvariant(&td.keeper)(td.ctx)
		require.True(t, broken)
		require.Contains(t, msg, keeper.NormalizedStateInvariantRoute)
		require.Contains(t, msg, "fractional balance")
		require.Contains(t, msg, "invalid")
	})

	t.Run("invalid remainder breaks invariant", func(t *testing.T) {
		td := newMockedTestData(t)
		td.ak.EXPECT().GetModuleAddress(types.ModuleName).Return(sdk.AccAddress{100}).Once()

		invalid := types.ConversionFactor()
		bz, err := invalid.Marshal()
		require.NoError(t, err)
		td.ctx.KVStore(td.storeKey).Set(types.RemainderBalanceKey, bz)

		msg, broken := keeper.NormalizedStateInvariant(&td.keeper)(td.ctx)
		require.True(t, broken)
		require.Contains(t, msg, keeper.NormalizedStateInvariantRoute)
		require.Contains(t, msg, "remainder amount is invalid")
	})
}

func TestReserveBackingInvariant(t *testing.T) {
	t.Run("backed reserve passes", func(t *testing.T) {
		td := newMockedTestData(t)
		reserveAddr := sdk.AccAddress{100}
		td.ak.EXPECT().GetModuleAddress(types.ModuleName).Return(reserveAddr).Times(2)
		td.bk.EXPECT().
			GetBalance(td.ctx, reserveAddr, types.IntegerCoinDenom()).
			Return(sdk.NewCoin(types.IntegerCoinDenom(), sdkmath.NewInt(1))).
			Once()

		td.keeper.SetFractionalBalance(td.ctx, sdk.AccAddress{1}, sdkmath.NewInt(400))
		td.keeper.SetFractionalBalance(td.ctx, sdk.AccAddress{2}, types.ConversionFactor().SubRaw(900))
		td.keeper.SetRemainderAmount(td.ctx, sdkmath.NewInt(500))

		msg, broken := keeper.ReserveBackingInvariant(&td.keeper)(td.ctx)
		require.False(t, broken)
		require.Empty(t, msg)
	})

	t.Run("underbacked reserve breaks invariant", func(t *testing.T) {
		td := newMockedTestData(t)
		reserveAddr := sdk.AccAddress{100}
		td.ak.EXPECT().GetModuleAddress(types.ModuleName).Return(reserveAddr).Times(2)
		td.bk.EXPECT().
			GetBalance(td.ctx, reserveAddr, types.IntegerCoinDenom()).
			Return(sdk.NewCoin(types.IntegerCoinDenom(), sdkmath.ZeroInt())).
			Once()

		td.keeper.SetFractionalBalance(td.ctx, sdk.AccAddress{1}, sdkmath.NewInt(1))
		td.keeper.SetRemainderAmount(td.ctx, types.ConversionFactor().SubRaw(1))

		msg, broken := keeper.ReserveBackingInvariant(&td.keeper)(td.ctx)
		require.True(t, broken)
		require.Contains(t, msg, keeper.ReserveBackingInvariantRoute)
		require.Contains(t, msg, "reserve backing mismatch")
	})
}
