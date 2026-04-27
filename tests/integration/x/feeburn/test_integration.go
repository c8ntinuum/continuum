package feeburn

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/evm"
	testconstants "github.com/cosmos/evm/testutil/constants"
	"github.com/cosmos/evm/testutil/integration/evm/network"
	testKeyring "github.com/cosmos/evm/testutil/keyring"
	precisebanktypes "github.com/cosmos/evm/x/precisebank/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
)

func TestBurnFeeCollectorBeforeDistribution(t *testing.T, create network.CreateEvmApp, options ...network.ConfigOption) {
	keyring := testKeyring.New(1)
	opts := []network.ConfigOption{
		network.WithChainID(testconstants.SixDecimalsChainID),
		network.WithPreFundedAccounts(keyring.GetAllAccAddrs()...),
	}
	opts = append(opts, options...)

	nw := network.NewUnitTestNetwork(create, opts...)
	require.NoError(t, nw.NextBlock(), "failed to advance to a clean post-init block")

	app := nw.App
	ctx := nw.GetContext()
	feeCollector := app.GetAccountKeeper().GetModuleAddress(authtypes.FeeCollectorName)
	require.NotNil(t, feeCollector, "fee collector module account must exist")
	require.True(t, effectiveBalances(app, ctx, feeCollector).Empty(), "expected fee collector to start empty")

	rewardsBefore := app.GetDistrKeeper().GetTotalRewards(ctx)
	feePoolBefore, err := app.GetDistrKeeper().FeePool.Get(ctx)
	require.NoError(t, err)

	integerDenom := precisebanktypes.IntegerCoinDenom()
	extendedDenom := precisebanktypes.ExtendedCoinDenom()

	integerSupplyBeforeSeed := app.GetBankKeeper().GetSupply(ctx, integerDenom)
	fooSupplyBeforeSeed := app.GetBankKeeper().GetSupply(ctx, "foo")

	passthroughCoins := sdk.NewCoins(
		sdk.NewInt64Coin("foo", 11),
		sdk.NewInt64Coin(integerDenom, 7),
	)
	err = app.GetBankKeeper().MintCoins(ctx, minttypes.ModuleName, passthroughCoins)
	require.NoError(t, err)
	err = app.GetBankKeeper().SendCoinsFromModuleToModule(ctx, minttypes.ModuleName, authtypes.FeeCollectorName, passthroughCoins)
	require.NoError(t, err)

	preciseAmount := precisebanktypes.ConversionFactor().AddRaw(500)
	preciseCoins := sdk.NewCoins(sdk.NewCoin(extendedDenom, preciseAmount))
	err = app.GetPreciseBankKeeper().MintCoins(ctx, minttypes.ModuleName, preciseCoins)
	require.NoError(t, err)
	err = app.GetPreciseBankKeeper().SendCoinsFromModuleToModule(ctx, minttypes.ModuleName, authtypes.FeeCollectorName, preciseCoins)
	require.NoError(t, err)

	expectedExtendedBalance := precisebanktypes.ConversionFactor().MulRaw(7).Add(preciseAmount)
	require.Equal(
		t,
		sdk.NewCoins(
			sdk.NewInt64Coin("foo", 11),
			sdk.NewCoin(extendedDenom, expectedExtendedBalance),
		).String(),
		effectiveBalances(app, ctx, feeCollector).String(),
	)

	integerSupplyBeforeBurn := app.GetBankKeeper().GetSupply(ctx, integerDenom)
	fooSupplyBeforeBurn := app.GetBankKeeper().GetSupply(ctx, "foo")

	require.NoError(t, nw.NextBlock(), "failed to advance burn block")

	ctx = nw.GetContext()
	require.True(t, effectiveBalances(app, ctx, feeCollector).Empty(), "expected fee collector to be empty after burn")

	integerSupplyAfterBurn := app.GetBankKeeper().GetSupply(ctx, integerDenom)
	fooSupplyAfterBurn := app.GetBankKeeper().GetSupply(ctx, "foo")

	require.Equal(
		t,
		integerSupplyBeforeBurn.Amount.Sub(integerSupplyAfterBurn.Amount),
		integerSupplyBeforeBurn.Amount.Sub(integerSupplyBeforeSeed.Amount),
		"unexpected integer supply burn delta",
	)
	require.Equal(
		t,
		fooSupplyBeforeBurn.Amount.Sub(fooSupplyAfterBurn.Amount),
		fooSupplyBeforeBurn.Amount.Sub(fooSupplyBeforeSeed.Amount),
		"unexpected passthrough denom burn delta",
	)

	rewardsAfter := app.GetDistrKeeper().GetTotalRewards(ctx)
	feePoolAfter, err := app.GetDistrKeeper().FeePool.Get(ctx)
	require.NoError(t, err)

	require.Equal(t, rewardsBefore.String(), rewardsAfter.String(), "distribution outstanding rewards should not increase")
	require.Equal(t, feePoolBefore.CommunityPool.String(), feePoolAfter.CommunityPool.String(), "community pool should not increase")
}

func effectiveBalances(app evm.EvmApp, ctx sdk.Context, addr sdk.AccAddress) sdk.Coins {
	balances := app.GetBankKeeper().GetAllBalances(ctx, addr)

	integerAmount := balances.AmountOf(precisebanktypes.IntegerCoinDenom())
	if integerAmount.IsPositive() {
		balances = balances.Sub(sdk.NewCoin(precisebanktypes.IntegerCoinDenom(), integerAmount))
	}

	extendedBalance := app.GetPreciseBankKeeper().GetBalance(ctx, addr, precisebanktypes.ExtendedCoinDenom())
	if extendedBalance.Amount.IsPositive() {
		balances = balances.Add(extendedBalance)
	}

	return balances
}
