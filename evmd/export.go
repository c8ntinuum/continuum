package evmd

import (
	"encoding/json"
	"fmt"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	storetypes "cosmossdk.io/store/types"

	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	disttypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// ExportAppStateAndValidators exports the state of the application for a genesis
// file.
func (app *EVMD) ExportAppStateAndValidators(forZeroHeight bool, jailAllowedAddrs []string, modulesToExport []string) (servertypes.ExportedApp, error) {
	// as if they could withdraw from the start of the next block
	ctx := app.NewContextLegacy(true, tmproto.Header{Height: app.LastBlockHeight()})

	// We export at last height + 1, because that's the height at which
	// CometBFT will start InitChain.
	height := app.LastBlockHeight() + 1
	if forZeroHeight {
		height = 0

		if err := app.prepForZeroHeightGenesis(ctx, jailAllowedAddrs); err != nil {
			return servertypes.ExportedApp{}, err
		}
	}

	genState, err := app.ModuleManager.ExportGenesisForModules(ctx, app.appCodec, modulesToExport)
	if err != nil {
		return servertypes.ExportedApp{}, err
	}

	// If the caller exported only a subset of modules from live state,
	// backfill every omitted module from the app's default genesis so the
	// final app_state remains complete for the new binary.
	if len(modulesToExport) > 0 {
		defaultGenState := app.DefaultGenesis()
		for moduleName, bz := range defaultGenState {
			if _, ok := genState[moduleName]; !ok {
				genState[moduleName] = bz
			}
		}
	}

	appState, err := json.MarshalIndent(genState, "", "  ")
	if err != nil {
		return servertypes.ExportedApp{}, err
	}

	validators, err := staking.WriteValidators(ctx, app.StakingKeeper)
	return servertypes.ExportedApp{
		AppState:        appState,
		Validators:      validators,
		Height:          height,
		ConsensusParams: app.GetConsensusParams(ctx),
	}, err
}

func cloneDecCoins(in sdk.DecCoins) sdk.DecCoins {
	if len(in) == 0 {
		return sdk.DecCoins{}
	}
	out := make(sdk.DecCoins, len(in))
	copy(out, in)
	return out
}

// pickSyntheticHistoricalRatio chooses a cumulative reward ratio for a missing
// historical period. Prefer the nearest later period if available, otherwise
// fall back to the nearest earlier period, and finally zero if the validator has
// no historical reward rows at all.
//
// This is intentionally conservative for zero-height export repair: using a
// later cumulative ratio tends to under-count old rewards instead of overpaying.
// Any leftover outstanding rewards are swept by prepForZeroHeightGenesis.
func pickSyntheticHistoricalRatio(
	history map[uint64]disttypes.ValidatorHistoricalRewards,
	target uint64,
) sdk.DecCoins {
	if len(history) == 0 {
		return sdk.DecCoins{}
	}

	if hr, ok := history[target]; ok {
		return cloneDecCoins(hr.CumulativeRewardRatio)
	}

	var (
		higherFound  bool
		higherPeriod uint64
		lowerFound   bool
		lowerPeriod  uint64
	)

	for period := range history {
		if period > target {
			if !higherFound || period < higherPeriod {
				higherFound = true
				higherPeriod = period
			}
		}
		if period < target {
			if !lowerFound || period > lowerPeriod {
				lowerFound = true
				lowerPeriod = period
			}
		}
	}

	if higherFound {
		return cloneDecCoins(history[higherPeriod].CumulativeRewardRatio)
	}
	if lowerFound {
		return cloneDecCoins(history[lowerPeriod].CumulativeRewardRatio)
	}

	return sdk.DecCoins{}
}

// repairDistrReferenceCounts rebuilds x/distribution historical reward reference
// counts from the actual objects that reference each period.
//
// In SDK v0.53.x, a historical rewards period can be referenced by:
//  1. validator current rewards at (current period - 1)
//  2. delegator starting info at PreviousPeriod
//  3. slash events at ValidatorPeriod
//
// The zero-height export path withdraws rewards and decrements these reference
// counts. If the stored counts are already wrong, export can panic with
// "cannot set negative reference count".
//
// If a referenced historical period is missing entirely, this function
// synthesizes a recovery row so that zero-height export can continue, but exact
// old reward accounting cannot be recovered once the referenced historical row
// has been lost.
// audit:B5: export-time repair path reviewed in audit/B5_RISK_MARKER_REVIEW.md.
func (app *EVMD) repairDistrReferenceCounts(ctx sdk.Context) error {
	type historicalRewardUpdate struct {
		val     sdk.ValAddress
		period  uint64
		rewards disttypes.ValidatorHistoricalRewards
	}

	type historicalRewardDelete struct {
		val    sdk.ValAddress
		period uint64
	}

	type historicalRewardCreate struct {
		val     sdk.ValAddress
		period  uint64
		rewards disttypes.ValidatorHistoricalRewards
	}

	expected := make(map[string]map[uint64]uint32)
	seen := make(map[string]map[uint64]bool)
	existing := make(map[string]map[uint64]disttypes.ValidatorHistoricalRewards)

	bump := func(val sdk.ValAddress, period uint64) error {
		key := string(val)
		if expected[key] == nil {
			expected[key] = make(map[uint64]uint32)
		}
		expected[key][period]++

		// SDK invariant: refcount is capped at 2.
		if expected[key][period] > 2 {
			return fmt.Errorf(
				"invalid expected reference count > 2 for validator=%s period=%d",
				val.String(),
				period,
			)
		}

		return nil
	}

	var walkErr error

	// 1) current rewards implicitly keep one reference to (current_period - 1)
	app.DistrKeeper.IterateValidatorCurrentRewards(ctx, func(
		val sdk.ValAddress,
		rewards disttypes.ValidatorCurrentRewards,
	) (stop bool) {
		if rewards.Period > 0 {
			if err := bump(val, rewards.Period-1); err != nil {
				walkErr = err
				return true
			}
		}
		return false
	})
	if walkErr != nil {
		return walkErr
	}

	// 2) each delegation starting record references PreviousPeriod
	app.DistrKeeper.IterateDelegatorStartingInfos(ctx, func(
		val sdk.ValAddress,
		_ sdk.AccAddress,
		info disttypes.DelegatorStartingInfo,
	) (stop bool) {
		if err := bump(val, info.PreviousPeriod); err != nil {
			walkErr = err
			return true
		}
		return false
	})
	if walkErr != nil {
		return walkErr
	}

	// 3) each slash event references ValidatorPeriod
	app.DistrKeeper.IterateValidatorSlashEvents(ctx, func(
		val sdk.ValAddress,
		_ uint64,
		ev disttypes.ValidatorSlashEvent,
	) (stop bool) {
		if err := bump(val, ev.ValidatorPeriod); err != nil {
			walkErr = err
			return true
		}
		return false
	})
	if walkErr != nil {
		return walkErr
	}

	// Collect mutations first, then apply them after iteration.
	var updates []historicalRewardUpdate
	var deletes []historicalRewardDelete

	app.DistrKeeper.IterateValidatorHistoricalRewards(ctx, func(
		val sdk.ValAddress,
		period uint64,
		rewards disttypes.ValidatorHistoricalRewards,
	) (stop bool) {
		key := string(val)
		if seen[key] == nil {
			seen[key] = make(map[uint64]bool)
		}
		seen[key][period] = true

		if existing[key] == nil {
			existing[key] = make(map[uint64]disttypes.ValidatorHistoricalRewards)
		}
		existing[key][period] = rewards

		want := expected[key][period]

		if want == 0 {
			deletes = append(deletes, historicalRewardDelete{
				val:    val,
				period: period,
			})
			return false
		}

		if rewards.ReferenceCount != want {
			rewards.ReferenceCount = want
			updates = append(updates, historicalRewardUpdate{
				val:     val,
				period:  period,
				rewards: rewards,
			})
		}

		return false
	})
	if walkErr != nil {
		return walkErr
	}

	var creates []historicalRewardCreate
	for key, periods := range expected {
		val := sdk.ValAddress([]byte(key))
		for period, want := range periods {
			if want == 0 {
				continue
			}
			if seen[key] != nil && seen[key][period] {
				continue
			}

			ratio := pickSyntheticHistoricalRatio(existing[key], period)
			creates = append(creates, historicalRewardCreate{
				val:    val,
				period: period,
				rewards: disttypes.ValidatorHistoricalRewards{
					CumulativeRewardRatio: ratio,
					ReferenceCount:        want,
				},
			})
		}
	}

	logger := app.Logger()

	for _, d := range deletes {
		logger.Info("repairDistrReferenceCounts: deleting unreferenced historical reward",
			"validator", d.val.String(), "period", d.period)
		if err := app.DistrKeeper.DeleteValidatorHistoricalReward(ctx, d.val, d.period); err != nil {
			return err
		}
	}

	for _, c := range creates {
		logger.Warn("repairDistrReferenceCounts: synthesizing missing historical reward (unsafe)",
			"validator", c.val.String(), "period", c.period, "refcount", c.rewards.ReferenceCount)
		if err := app.DistrKeeper.SetValidatorHistoricalRewards(ctx, c.val, c.period, c.rewards); err != nil {
			return err
		}
	}

	for _, u := range updates {
		logger.Info("repairDistrReferenceCounts: fixing reference count",
			"validator", u.val.String(), "period", u.period, "new_refcount", u.rewards.ReferenceCount)
		if err := app.DistrKeeper.SetValidatorHistoricalRewards(ctx, u.val, u.period, u.rewards); err != nil {
			return err
		}
	}

	if len(deletes)+len(creates)+len(updates) > 0 {
		logger.Info("repairDistrReferenceCounts: repair complete",
			"deleted", len(deletes), "synthesized", len(creates), "refcount_fixed", len(updates))
	} else {
		logger.Info("repairDistrReferenceCounts: no repairs needed")
	}

	return nil
}

// prepare for fresh start at zero height
// NOTE zero height genesis is a temporary feature which will be deprecated
//
//	in favour of export at a block height
func (app *EVMD) prepForZeroHeightGenesis(ctx sdk.Context, jailAllowedAddrs []string) error {
	applyAllowedAddrs := len(jailAllowedAddrs) > 0

	// check if there is an allowed address list
	allowedAddrsMap := make(map[string]bool)
	for _, addr := range jailAllowedAddrs {
		_, err := sdk.ValAddressFromBech32(addr)
		if err != nil {
			return fmt.Errorf("invalid jail-allowed validator address %q: %w", addr, err)
		}
		allowedAddrsMap[addr] = true
	}

	/* Handle fee distribution state. */

	// The standard SDK zero-height flow (withdraw commission → withdraw rewards
	// → reinitialize) relies on x/distribution historical-reward reference counts
	// being correct. When the live store is corrupt (missing or zero-refcount
	// entries), the withdrawal path panics in decrementReferenceCount.
	//
	// Instead we perform a "nuclear" distribution reset:
	//   1. Sweep all outstanding rewards to the community pool.
	//   2. Delete ALL per-validator / per-delegator distribution state.
	//   3. Reinitialize validators and delegations from scratch at height 0.
	//
	// Trade-off: unrealised rewards / commission stay in the distribution module
	// account rather than being sent to individual accounts. For a zero-height
	// genesis restart this is acceptable — the community pool reflects the
	// value, and individual balances are whatever they are at export time.

	logger := app.Logger()
	logger.Info("prepForZeroHeightGenesis: nuclear distribution reset")

	// 1. Sweep outstanding rewards to community pool.
	feePool, err := app.DistrKeeper.FeePool.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to get fee pool: %w", err)
	}

	var outstandingVals []sdk.ValAddress
	app.DistrKeeper.IterateValidatorOutstandingRewards(ctx,
		func(val sdk.ValAddress, rewards disttypes.ValidatorOutstandingRewards) (stop bool) {
			feePool.CommunityPool = feePool.CommunityPool.Add(rewards.Rewards...)
			outstandingVals = append(outstandingVals, val)
			return false
		})

	if err := app.DistrKeeper.FeePool.Set(ctx, feePool); err != nil {
		return fmt.Errorf("failed to set fee pool: %w", err)
	}

	// 2a. Delete outstanding rewards, accumulated commission, current rewards.
	for _, val := range outstandingVals {
		_ = app.DistrKeeper.DeleteValidatorOutstandingRewards(ctx, val)
	}

	var currentRewardVals []sdk.ValAddress
	app.DistrKeeper.IterateValidatorCurrentRewards(ctx,
		func(val sdk.ValAddress, _ disttypes.ValidatorCurrentRewards) (stop bool) {
			currentRewardVals = append(currentRewardVals, val)
			return false
		})
	for _, val := range currentRewardVals {
		_ = app.DistrKeeper.DeleteValidatorCurrentRewards(ctx, val)
	}

	var commissionVals []sdk.ValAddress
	app.DistrKeeper.IterateValidatorAccumulatedCommissions(ctx,
		func(val sdk.ValAddress, _ disttypes.ValidatorAccumulatedCommission) (stop bool) {
			commissionVals = append(commissionVals, val)
			return false
		})
	for _, val := range commissionVals {
		_ = app.DistrKeeper.DeleteValidatorAccumulatedCommission(ctx, val)
	}

	// 2b. Delete all historical rewards and slash events.
	app.DistrKeeper.DeleteAllValidatorHistoricalRewards(ctx)
	app.DistrKeeper.DeleteAllValidatorSlashEvents(ctx)

	// 2c. Delete all delegator starting infos.
	type startInfoKey struct {
		val sdk.ValAddress
		del sdk.AccAddress
	}
	var siKeys []startInfoKey
	app.DistrKeeper.IterateDelegatorStartingInfos(ctx,
		func(val sdk.ValAddress, del sdk.AccAddress, _ disttypes.DelegatorStartingInfo) (stop bool) {
			siKeys = append(siKeys, startInfoKey{val, del})
			return false
		})
	for _, k := range siKeys {
		_ = app.DistrKeeper.DeleteDelegatorStartingInfo(ctx, k.val, k.del)
	}

	logger.Info("prepForZeroHeightGenesis: distribution state cleared",
		"outstanding_swept", len(outstandingVals),
		"current_rewards_deleted", len(currentRewardVals),
		"commission_deleted", len(commissionVals),
		"starting_infos_deleted", len(siKeys))

	// 3. Reinitialize all validators at height 0 (creates period 0, current
	//    period 1, empty outstanding / commission for each validator).
	//    Track which validators initialised successfully so we can skip their
	//    delegations if they didn't.
	height := ctx.BlockHeight()
	ctx = ctx.WithBlockHeight(0)

	initedVals := make(map[string]bool)
	if err := app.StakingKeeper.IterateValidators(ctx, func(_ int64, val stakingtypes.ValidatorI) (stop bool) {
		valAddr := sdk.ValAddress(val.GetOperator())
		if err := app.DistrKeeper.Hooks().AfterValidatorCreated(ctx, valAddr); err != nil {
			logger.Warn("prepForZeroHeightGenesis: AfterValidatorCreated failed, skipping validator",
				"validator", valAddr.String(), "err", err)
			return false // continue to next validator
		}
		initedVals[valAddr.String()] = true
		return false
	}); err != nil {
		return err
	}

	// 4. Reinitialize all delegations (each gets a fresh starting info
	//    referencing the period just created by the validator init above).
	//    Skip delegations to validators that failed initialization.
	dels, err := app.StakingKeeper.GetAllDelegations(ctx)
	if err != nil {
		return err
	}

	for _, del := range dels {
		if !initedVals[del.ValidatorAddress] {
			logger.Warn("prepForZeroHeightGenesis: skipping delegation to uninitialised validator",
				"delegator", del.DelegatorAddress, "validator", del.ValidatorAddress)
			continue
		}

		valAddr, err := sdk.ValAddressFromBech32(del.ValidatorAddress)
		if err != nil {
			return fmt.Errorf("invalid validator address %q: %w", del.ValidatorAddress, err)
		}
		delAddr := sdk.MustAccAddressFromBech32(del.DelegatorAddress)

		if err := app.DistrKeeper.Hooks().BeforeDelegationCreated(ctx, delAddr, valAddr); err != nil {
			return fmt.Errorf("error while incrementing period: %w", err)
		}

		if err := app.DistrKeeper.Hooks().AfterDelegationModified(ctx, delAddr, valAddr); err != nil {
			return fmt.Errorf("error while creating delegation period record: %w", err)
		}
	}

	// reset context height
	ctx = ctx.WithBlockHeight(height)

	/* Handle staking state. */

	// iterate through redelegations, reset creation height
	var iterErr error
	if err := app.StakingKeeper.IterateRedelegations(ctx, func(_ int64, red stakingtypes.Redelegation) (stop bool) {
		for i := range red.Entries {
			red.Entries[i].CreationHeight = 0
		}
		if iterErr = app.StakingKeeper.SetRedelegation(ctx, red); iterErr != nil {
			return true
		}
		return false
	}); err != nil {
		return err
	}

	if iterErr != nil {
		return iterErr
	}

	// iterate through unbonding delegations, reset creation height
	if err := app.StakingKeeper.IterateUnbondingDelegations(ctx, func(_ int64, ubd stakingtypes.UnbondingDelegation) (stop bool) {
		for i := range ubd.Entries {
			ubd.Entries[i].CreationHeight = 0
		}
		if iterErr = app.StakingKeeper.SetUnbondingDelegation(ctx, ubd); iterErr != nil {
			return true
		}
		return false
	}); err != nil {
		return err
	}

	if iterErr != nil {
		return iterErr
	}

	// Iterate through validators by power descending, reset bond heights, and
	// update bond intra-tx counters.
	store := ctx.KVStore(app.GetKey(stakingtypes.StoreKey))
	iter := storetypes.KVStoreReversePrefixIterator(store, stakingtypes.ValidatorsKey)
	counter := int16(0)

	for ; iter.Valid(); iter.Next() {
		addr := sdk.ValAddress(stakingtypes.AddressFromValidatorsKey(iter.Key()))
		validator, err := app.StakingKeeper.GetValidator(ctx, addr)
		if err != nil {
			return fmt.Errorf("expected validator %s not found. Error: %w", addr, err)
		}

		validator.UnbondingHeight = 0
		if applyAllowedAddrs && !allowedAddrsMap[addr.String()] {
			validator.Jailed = true
		}

		if err = app.StakingKeeper.SetValidator(ctx, validator); err != nil {
			return err
		}
		counter++
	}

	if err := iter.Close(); err != nil {
		app.Logger().Error("error while closing the key-value store reverse prefix iterator: ", err)
		return nil
	}

	_, err = app.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	if err != nil {
		return fmt.Errorf("failed to apply validator set updates: %w", err)
	}

	/* Handle slashing state. */

	// reset start height on signing infos
	if err := app.SlashingKeeper.IterateValidatorSigningInfos(
		ctx,
		func(addr sdk.ConsAddress, info slashingtypes.ValidatorSigningInfo) (stop bool) {
			info.StartHeight = 0
			if iterErr = app.SlashingKeeper.SetValidatorSigningInfo(ctx, addr, info); iterErr != nil {
				return true
			}
			return false
		},
	); err != nil {
		return err
	}

	if iterErr != nil {
		return iterErr
	}

	return nil
}
