package keeper

import (
	"fmt"

	"github.com/cosmos/evm/x/precisebank/types"

	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store/prefix"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	NormalizedStateInvariantRoute = "normalized-state"
	ReserveBackingInvariantRoute  = "reserve-backing"
)

type invariantSnapshot struct {
	totalFractional sdkmath.Int
	remainder       sdkmath.Int
}

// RegisterInvariants registers all x/precisebank invariants.
//
//nolint:staticcheck // x/crisis-backed invariant registration is intentional for startup/runtime assertions.
func RegisterInvariants(ir sdk.InvariantRegistry, k *Keeper) {
	ir.RegisterRoute(types.ModuleName, NormalizedStateInvariantRoute, NormalizedStateInvariant(k))
	ir.RegisterRoute(types.ModuleName, ReserveBackingInvariantRoute, ReserveBackingInvariant(k))
}

// AllInvariants runs all x/precisebank invariants.
//
//nolint:staticcheck // x/crisis-backed invariant registration is intentional for startup/runtime assertions.
func AllInvariants(k *Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		if res, stop := NormalizedStateInvariant(k)(ctx); stop {
			return res, stop
		}

		return ReserveBackingInvariant(k)(ctx)
	}
}

// NormalizedStateInvariant checks that all persisted fractional balances and
// the module remainder stay within the normalized fractional range.
//
//nolint:staticcheck // x/crisis-backed invariant registration is intentional for startup/runtime assertions.
func NormalizedStateInvariant(k *Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		_, msg, broken := readInvariantSnapshot(ctx, k)
		if broken {
			return sdk.FormatInvariant(types.ModuleName, NormalizedStateInvariantRoute, msg), true
		}

		return "", false
	}
}

// ReserveBackingInvariant checks that the x/bank reserve balance fully backs
// the total tracked fractional state.
//
//nolint:staticcheck // x/crisis-backed invariant registration is intentional for startup/runtime assertions.
func ReserveBackingInvariant(k *Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		snapshot, msg, broken := readInvariantSnapshot(ctx, k)
		if broken {
			return sdk.FormatInvariant(types.ModuleName, ReserveBackingInvariantRoute, msg), true
		}

		reserveAddr := k.ak.GetModuleAddress(types.ModuleName)
		if reserveAddr == nil {
			return sdk.FormatInvariant(
				types.ModuleName,
				ReserveBackingInvariantRoute,
				fmt.Sprintf("module account %s is not registered", types.ModuleName),
			), true
		}

		requiredBacking := snapshot.totalFractional.Add(snapshot.remainder)
		reserveBalance := k.bk.GetBalance(ctx, reserveAddr, types.IntegerCoinDenom())
		availableBacking := reserveBalance.Amount.Mul(types.ConversionFactor())

		if !requiredBacking.Equal(availableBacking) {
			return sdk.FormatInvariant(
				types.ModuleName,
				ReserveBackingInvariantRoute,
				fmt.Sprintf(
					"reserve backing mismatch: reserve balance %s requires backing for %s%s but tracked fractional state requires %s%s",
					reserveBalance,
					availableBacking,
					types.ExtendedCoinDenom(),
					requiredBacking,
					types.ExtendedCoinDenom(),
				),
			), true
		}

		return "", false
	}
}

func readInvariantSnapshot(ctx sdk.Context, k *Keeper) (invariantSnapshot, string, bool) {
	snapshot := invariantSnapshot{
		totalFractional: sdkmath.ZeroInt(),
		remainder:       sdkmath.ZeroInt(),
	}

	reserveAddr := k.ak.GetModuleAddress(types.ModuleName)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.FractionalBalancePrefix)
	iterator := store.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		addr := sdk.AccAddress(iterator.Key())
		if addr.Empty() {
			return snapshot, "fractional balance entry uses an empty address key", true
		}
		if reserveAddr != nil && addr.Equals(reserveAddr) {
			return snapshot, fmt.Sprintf("reserve module account %s has a fractional balance entry", reserveAddr.String()), true
		}

		var amount sdkmath.Int
		if err := amount.Unmarshal(iterator.Value()); err != nil {
			return snapshot, fmt.Sprintf("failed to unmarshal fractional balance for address key %X: %v", iterator.Key(), err), true
		}
		if err := types.ValidateFractionalAmount(amount); err != nil {
			return snapshot, fmt.Sprintf("fractional balance for address key %X is invalid: %v", iterator.Key(), err), true
		}

		snapshot.totalFractional = snapshot.totalFractional.Add(amount)
	}

	bz := ctx.KVStore(k.storeKey).Get(types.RemainderBalanceKey)
	if bz != nil {
		if err := snapshot.remainder.Unmarshal(bz); err != nil {
			return snapshot, fmt.Sprintf("failed to unmarshal remainder amount: %v", err), true
		}
		if err := types.ValidateFractionalAmount(snapshot.remainder); err != nil {
			return snapshot, fmt.Sprintf("remainder amount is invalid: %v", err), true
		}
	}

	return snapshot, "", false
}
