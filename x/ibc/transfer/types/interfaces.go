package types

import (
	"context"

	erc20types "github.com/cosmos/evm/x/erc20/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AccountKeeper defines the expected interface needed to retrieve account info.
type AccountKeeper interface {
	transfertypes.AccountKeeper
	GetAccount(context.Context, sdk.AccAddress) sdk.AccountI
}

// BankKeeper defines the expected interface needed to check balances and send coins.
type BankKeeper interface {
	transfertypes.BankKeeper
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
}

// ChannelKeeper defines the expected transfer channel keeper interface.
type ChannelKeeper interface {
	transfertypes.ChannelKeeper
	porttypes.ICS4Wrapper
}

// ERC20Keeper defines the expected ERC20 keeper interface for supporting
// ERC20 token transfers via IBC.
type ERC20Keeper interface {
	IsERC20Enabled(ctx sdk.Context) bool
	GetTokenPairID(ctx sdk.Context, token string) []byte
	GetTokenPair(ctx sdk.Context, id []byte) (erc20types.TokenPair, bool)
	ConvertERC20(ctx context.Context, msg *erc20types.MsgConvertERC20) (*erc20types.MsgConvertERC20Response, error)
}

// IbcBreakerKeeper defines the expected IBC breaker keeper interface.
type IbcBreakerKeeper interface {
	GetIbcAvailable(ctx sdk.Context) bool
}

// CircuitKeeper defines the expected circuit breaker keeper interface.
type CircuitKeeper interface {
	GetSystemAvailable(ctx sdk.Context) bool
}
