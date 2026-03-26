package types

import (
	"context"

	"cosmossdk.io/core/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type AccountKeeper interface {
	AddressCodec() address.Codec
	GetModuleAccount(ctx context.Context, moduleName string) sdk.ModuleAccountI
	GetModuleAddress(moduleName string) sdk.AccAddress
	GetSequence(context.Context, sdk.AccAddress) (uint64, error)
}
type BankKeeper interface {
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}
