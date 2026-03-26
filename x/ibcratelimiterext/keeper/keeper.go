package keeper

import (
	"github.com/cosmos/evm/x/ibcratelimiterext/types"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Keeper struct {
	cdc       codec.BinaryCodec
	storeKey  storetypes.StoreKey
	authority sdk.AccAddress
}

func NewKeeper(cdc codec.BinaryCodec, storeKey storetypes.StoreKey, authority sdk.AccAddress) Keeper {
	if err := sdk.VerifyAddressFormat(authority); err != nil {
		panic(err)
	}
	return Keeper{cdc: cdc, storeKey: storeKey, authority: authority}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", types.ModuleName)
}

func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyParams)
	if bz == nil {
		return types.DefaultParams()
	}

	var params types.Params
	k.cdc.MustUnmarshal(bz, &params)
	return params
}

func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.KeyParams, k.cdc.MustMarshal(&params))
}

func (k Keeper) IsWhitelisted(ctx sdk.Context, addr sdk.AccAddress) bool {
	params := k.GetParams(ctx)
	addrStr := addr.String()
	for _, allowed := range params.Whitelist {
		if allowed == addrStr {
			return true
		}
	}
	return false
}
