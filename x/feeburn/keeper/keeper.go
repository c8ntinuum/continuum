package keeper

import (
	"context"
	"fmt"

	"github.com/cosmos/evm/x/precisebank/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

type AccountKeeper interface {
	GetModuleAddress(name string) sdk.AccAddress
}

type BankKeeper interface {
	GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins
}

type PreciseBankKeeper interface {
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
}

type Keeper struct {
	accountKeeper     AccountKeeper
	bankKeeper        BankKeeper
	preciseBankKeeper PreciseBankKeeper
}

func NewKeeper(accountKeeper AccountKeeper, bankKeeper BankKeeper, preciseBankKeeper PreciseBankKeeper) Keeper {
	return Keeper{
		accountKeeper:     accountKeeper,
		bankKeeper:        bankKeeper,
		preciseBankKeeper: preciseBankKeeper,
	}
}

func (k Keeper) BeginBlock(ctx context.Context) error {
	feeCollector := k.accountKeeper.GetModuleAddress(authtypes.FeeCollectorName)
	if feeCollector == nil {
		panic(fmt.Sprintf("%s module account has not been set", authtypes.FeeCollectorName))
	}

	burnCoins := k.bankKeeper.GetAllBalances(ctx, feeCollector)

	integerAmount := burnCoins.AmountOf(types.IntegerCoinDenom())
	if integerAmount.IsPositive() {
		burnCoins = burnCoins.Sub(sdk.NewCoin(types.IntegerCoinDenom(), integerAmount))
	}

	extendedBalance := k.preciseBankKeeper.GetBalance(ctx, feeCollector, types.ExtendedCoinDenom())
	if extendedBalance.Amount.IsPositive() {
		burnCoins = burnCoins.Add(extendedBalance)
	}

	if burnCoins.Empty() {
		return nil
	}

	return k.preciseBankKeeper.BurnCoins(ctx, authtypes.FeeCollectorName, burnCoins)
}
