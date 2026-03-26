package keeper

import (
	"github.com/cosmos/evm/x/ibc/transfer/types"
	ibctransferkeeper "github.com/cosmos/ibc-go/v10/modules/apps/transfer/keeper"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"

	"cosmossdk.io/core/address"
	corestore "cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/codec"
)

// Keeper defines the modified IBC transfer keeper that embeds the original one.
// It also contains the bank keeper and the erc20 keeper to support ERC20 tokens
// to be sent via IBC.
type Keeper struct {
	*ibctransferkeeper.Keeper
	bankKeeper       types.BankKeeper
	erc20Keeper      types.ERC20Keeper
	accountKeeper    types.AccountKeeper
	ibcBreakerKeeper types.IbcBreakerKeeper
}

// NewKeeper creates a new IBC transfer Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	addressCodec address.Codec,
	storeService corestore.KVStoreService,
	channelKeeper types.ChannelKeeper,
	msgRouter transfertypes.MessageRouter,
	authKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	erc20Keeper types.ERC20Keeper,
	authority string,
	ibcBreakerKeeper types.IbcBreakerKeeper,
) Keeper {
	// create the original IBC transfer keeper for embedding
	transferKeeper := ibctransferkeeper.NewKeeper(
		cdc,
		storeService,
		nil,
		channelKeeper,
		channelKeeper,
		msgRouter,
		authKeeper,
		bankKeeper,
		authority,
	)
	transferKeeper.SetAddressCodec(addressCodec)

	return Keeper{
		Keeper:           &transferKeeper,
		bankKeeper:       bankKeeper,
		erc20Keeper:      erc20Keeper,
		accountKeeper:    authKeeper,
		ibcBreakerKeeper: ibcBreakerKeeper,
	}
}
