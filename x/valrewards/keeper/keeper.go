package keeper

import (
	"encoding/hex"
	"errors"
	"fmt"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	vrtypes "github.com/cosmos/evm/x/valrewards/types"
	vrutils "github.com/cosmos/evm/x/valrewards/utils"
)

type Keeper struct {
	cdc           codec.BinaryCodec
	storeKey      storetypes.StoreKey
	authority     sdk.AccAddress
	stakingKeeper vrtypes.StakingKeeper
	accountKeeper vrtypes.AccountKeeper
	bankKeeper    vrtypes.BankKeeper
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	authority sdk.AccAddress,
	stakingKeeper vrtypes.StakingKeeper,
	accountKeeper vrtypes.AccountKeeper,
	bankKeeper vrtypes.BankKeeper,
) Keeper {
	if err := sdk.VerifyAddressFormat(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %v", err)) //nolint:halt // Constructor guard: valrewards authority wiring must be valid at boot.
	}

	return Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		authority:     authority,
		stakingKeeper: stakingKeeper,
		accountKeeper: accountKeeper,
		bankKeeper:    bankKeeper,
	}
}

func (k Keeper) BeginBlocker(ctx sdk.Context) error {
	if ctx.BlockHeight() <= 1 {
		return nil
	}

	currentSettings := k.GetCurrentRewardSettings(ctx)
	epochState := k.GetEpochState(ctx)

	k.processValidatorsPoints(ctx, epochState.CurrentEpoch, currentSettings)

	epochState.BlocksIntoCurrentEpoch++
	if epochState.BlocksIntoCurrentEpoch >= currentSettings.BlocksInEpoch {
		k.processValidatorsRewards(ctx, epochState.CurrentEpoch, currentSettings)

		nextSettings := k.GetNextRewardSettings(ctx)
		if err := nextSettings.Validate(); err != nil {
			fallbackSettings := currentSettings
			fallbackLabel := "current settings"
			if fallbackErr := fallbackSettings.Validate(); fallbackErr != nil {
				ctx.Logger().Error(
					"valrewards: current reward settings are also invalid, resetting to defaults",
					"epoch", epochState.CurrentEpoch+1,
					"err", fallbackErr,
				)
				fallbackSettings = vrtypes.DefaultRewardSettings()
				fallbackLabel = "default settings"
			}

			ctx.Logger().Error(
				"valrewards: invalid staged reward settings, applying fallback settings",
				"epoch", epochState.CurrentEpoch+1,
				"fallback", fallbackLabel,
				"err", err,
			)
			nextSettings = fallbackSettings
		}
		epochState.CurrentEpoch++
		epochState.BlocksIntoCurrentEpoch = 0

		k.SetEpochState(ctx, epochState)
		k.SetCurrentRewardSettings(ctx, nextSettings)
		k.SetNextRewardSettings(ctx, nextSettings)
		return nil
	}

	k.SetEpochState(ctx, epochState)
	return nil
}

func (k Keeper) processValidatorsRewards(ctx sdk.Context, completedEpoch uint64, settings vrtypes.RewardSettings) {
	requiredRewards, err := settings.RewardsCoin()
	if err != nil {
		ctx.Logger().Error(
			"valrewards: invalid reward settings, skipping epoch payout",
			"epoch", completedEpoch,
			"err", err,
		)
		k.SetEpochToPay(ctx, completedEpoch+1)
		return
	}

	epochValidatorsToPay := k.GetEpochValidatorsPoints(ctx, completedEpoch)
	var epochTotalPoints uint64
	for _, epochValidatorToPay := range epochValidatorsToPay {
		epochTotalPoints += epochValidatorToPay.EpochPoints
	}

	if epochTotalPoints == 0 {
		k.SetEpochToPay(ctx, completedEpoch+1)
		return
	}

	for _, epochValidatorToPay := range epochValidatorsToPay {
		paymentFraction := math.LegacyNewDec(int64(epochValidatorToPay.EpochPoints)).QuoTruncate(math.LegacyNewDec(int64(epochTotalPoints)))
		toPayAmount := math.LegacyNewDecFromInt(requiredRewards.Amount).Mul(paymentFraction).TruncateInt()
		toPayCoin := sdk.NewCoin(evmtypes.DefaultEVMDenom, toPayAmount)
		k.SetValidatorOutstandingReward(ctx, completedEpoch, epochValidatorToPay.ValidatorAddress, toPayCoin)
	}

	k.SetEpochToPay(ctx, completedEpoch+1)
}

func (k Keeper) processValidatorsPoints(ctx sdk.Context, currentEpoch uint64, settings vrtypes.RewardSettings) {
	if settings.RewardingPaused {
		return
	}

	var prevBlockTotalPower int64
	for _, voteInfo := range ctx.VoteInfos() {
		prevBlockTotalPower += voteInfo.Validator.Power
	}
	if prevBlockTotalPower == 0 {
		return
	}

	proposerAddress := sdk.ConsAddress(ctx.BlockHeader().ProposerAddress).String()
	for _, voteInfo := range ctx.VoteInfos() {
		validator, err := k.stakingKeeper.ValidatorByConsAddr(ctx, voteInfo.Validator.Address)
		if err != nil {
			continue
		}

		powerFraction := math.LegacyNewDec(voteInfo.Validator.Power).QuoTruncate(math.LegacyNewDec(prevBlockTotalPower))
		validatorPoints := uint64(powerFraction.MulInt64(100).TruncateInt64())
		k.AddValidatorRewardPoints(ctx, currentEpoch, validator.GetOperator(), validatorPoints)

		if sdk.ConsAddress(voteInfo.Validator.Address).String() == proposerAddress {
			k.AddValidatorRewardPoints(ctx, currentEpoch, validator.GetOperator(), vrtypes.PROPOSER_BONUS_POINTS)
		}
	}
}

func (k Keeper) GetParams(ctx sdk.Context) vrtypes.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(vrtypes.GetParamsKey())
	if bz == nil {
		return vrtypes.DefaultParams()
	}

	var params vrtypes.Params
	k.cdc.MustUnmarshal(bz, &params)
	return params
}

func (k Keeper) SetParams(ctx sdk.Context, params vrtypes.Params) {
	store := ctx.KVStore(k.storeKey)
	store.Set(vrtypes.GetParamsKey(), k.cdc.MustMarshal(&params))
}

func (k Keeper) IsWhitelisted(ctx sdk.Context, addr sdk.AccAddress) bool {
	params := k.GetParams(ctx)
	for _, whitelistAddr := range params.Whitelist {
		if whitelistAddr == addr.String() {
			return true
		}
	}

	return false
}

func (k Keeper) GetCurrentRewardSettings(ctx sdk.Context) vrtypes.RewardSettings {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(vrtypes.GetCurrentRewardSettingsKey())
	if bz == nil {
		return vrtypes.DefaultRewardSettings()
	}

	var settings vrtypes.RewardSettings
	k.cdc.MustUnmarshal(bz, &settings)
	return settings
}

func (k Keeper) SetCurrentRewardSettings(ctx sdk.Context, settings vrtypes.RewardSettings) {
	if err := settings.Validate(); err != nil {
		panic(fmt.Errorf("invalid current reward settings: %w", err)) //nolint:halt // Internal invariant: keeper callers must validate reward settings before persisting them.
	}

	store := ctx.KVStore(k.storeKey)
	store.Set(vrtypes.GetCurrentRewardSettingsKey(), k.cdc.MustMarshal(&settings))
}

func (k Keeper) GetNextRewardSettings(ctx sdk.Context) vrtypes.RewardSettings {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(vrtypes.GetNextRewardSettingsKey())
	if bz == nil {
		return k.GetCurrentRewardSettings(ctx)
	}

	var settings vrtypes.RewardSettings
	k.cdc.MustUnmarshal(bz, &settings)
	return settings
}

func (k Keeper) SetNextRewardSettings(ctx sdk.Context, settings vrtypes.RewardSettings) {
	if err := settings.Validate(); err != nil {
		panic(fmt.Errorf("invalid next reward settings: %w", err)) //nolint:halt // Internal invariant: keeper callers must validate staged reward settings before persisting them.
	}

	store := ctx.KVStore(k.storeKey)
	store.Set(vrtypes.GetNextRewardSettingsKey(), k.cdc.MustMarshal(&settings))
}

func (k Keeper) GetEpochState(ctx sdk.Context) vrtypes.EpochState {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(vrtypes.GetEpochStateKey())
	if bz == nil {
		return k.legacyEpochStateFromHeight(ctx)
	}

	var state vrtypes.EpochState
	k.cdc.MustUnmarshal(bz, &state)
	return state
}

func (k Keeper) SetEpochState(ctx sdk.Context, state vrtypes.EpochState) {
	store := ctx.KVStore(k.storeKey)
	store.Set(vrtypes.GetEpochStateKey(), k.cdc.MustMarshal(&state))
}

func (k Keeper) legacyEpochStateFromHeight(ctx sdk.Context) vrtypes.EpochState {
	assignedBlocks := ctx.BlockHeight() - 2
	if assignedBlocks < 0 {
		assignedBlocks = 0
	}

	currentSettings := k.GetCurrentRewardSettings(ctx)
	blocksInEpoch := currentSettings.BlocksInEpoch
	if blocksInEpoch <= 0 {
		blocksInEpoch = vrtypes.BLOCKS_IN_EPOCH
	}

	return vrtypes.EpochState{
		CurrentEpoch:           uint64(assignedBlocks / blocksInEpoch),
		BlocksIntoCurrentEpoch: assignedBlocks % blocksInEpoch,
	}
}

func (k Keeper) GetEpochToPay(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	key := vrtypes.GetEpochToPayKey()
	bz := store.Get(key)
	return vrutils.DecodeEpochToPay(bz)
}

func (k Keeper) SetEpochToPay(ctx sdk.Context, nextEpoch uint64) {
	store := ctx.KVStore(k.storeKey)
	key := vrtypes.GetEpochToPayKey()
	bz := vrutils.EncodeEpochToPay(nextEpoch)
	store.Set(key, bz)
}

func (k Keeper) SetValidatorOutstandingReward(ctx sdk.Context, epoch uint64, validatorAddress string, amount sdk.Coin) {
	validatorAddressBytes := []byte(validatorAddress)
	store := ctx.KVStore(k.storeKey)
	key := vrtypes.GetEpochValidatorOutstandingKey(epoch, validatorAddressBytes)
	bz := vrutils.EncodeCoin(amount)
	store.Set(key, bz)
}

func (k Keeper) GetValidatorOutstandingReward(ctx sdk.Context, epoch uint64, validatorAddress string) sdk.Coin {
	validatorAddressBytes := []byte(validatorAddress)
	store := ctx.KVStore(k.storeKey)
	key := vrtypes.GetEpochValidatorOutstandingKey(epoch, validatorAddressBytes)
	bz := store.Get(key)
	return vrutils.DecodeCoin(bz)
}

func (k Keeper) iterateValidatorsOutstanding(ctx sdk.Context, epoch uint64, cb func(validatorAddress string, amount sdk.Coin) (stop bool)) {
	store := ctx.KVStore(k.storeKey)
	allEpochValidatorsKey := vrtypes.GetEpochValidatorOutstandingListKey(epoch)
	iterator := storetypes.KVStorePrefixIterator(store, allEpochValidatorsKey)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		validatorAddressBytes := key[len(allEpochValidatorsKey):]
		validatorAddress := string(validatorAddressBytes)
		amount := vrutils.DecodeCoin(iterator.Value())
		if cb(validatorAddress, amount) {
			break
		}
	}
}

func (k Keeper) GetEpochValidatorsOutstandings(ctx sdk.Context, epoch uint64) []*vrtypes.EpochValidatorOutstanding {
	var result []*vrtypes.EpochValidatorOutstanding
	k.iterateValidatorsOutstanding(ctx, epoch, func(validatorAddress string, amount sdk.Coin) bool {
		result = append(result, &vrtypes.EpochValidatorOutstanding{
			ValidatorAddress: validatorAddress,
			Amount:           amount,
		})
		return false
	})
	return result
}

func (k Keeper) AddValidatorRewardPoints(ctx sdk.Context, epoch uint64, validatorAddress string, points uint64) {
	validatorAddressBytes := []byte(validatorAddress)
	store := ctx.KVStore(k.storeKey)
	key := vrtypes.GetEpochValidatorPointsKey(epoch, validatorAddressBytes)
	var currentPoints uint64
	if bz := store.Get(key); bz != nil {
		currentPoints = vrutils.DecodePoints(bz)
	}
	store.Set(key, vrutils.EncodePoints(currentPoints+points))
}

func (k Keeper) SetValidatorRewardPoints(ctx sdk.Context, epoch uint64, validatorAddress string, points uint64) {
	validatorAddressBytes := []byte(validatorAddress)
	store := ctx.KVStore(k.storeKey)
	key := vrtypes.GetEpochValidatorPointsKey(epoch, validatorAddressBytes)
	store.Set(key, vrutils.EncodePoints(points))
}

func (k Keeper) GetValidatorPoints(ctx sdk.Context, epoch uint64, validatorAddress string) uint64 {
	validatorAddressBytes := []byte(validatorAddress)
	store := ctx.KVStore(k.storeKey)
	key := vrtypes.GetEpochValidatorPointsKey(epoch, validatorAddressBytes)
	var currentPoints uint64
	if bz := store.Get(key); bz != nil {
		currentPoints = vrutils.DecodePoints(bz)
	}
	return currentPoints
}

func (k Keeper) IterateValidatorsPoints(ctx sdk.Context, epoch uint64, cb func(validatorAddress string, points uint64) (stop bool)) {
	store := ctx.KVStore(k.storeKey)
	allEpochValidatorsKey := vrtypes.GetEpochValidatorPointsListKey(epoch)
	iterator := storetypes.KVStorePrefixIterator(store, allEpochValidatorsKey)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		validatorAddressBytes := key[len(allEpochValidatorsKey):]
		validatorAddress := string(validatorAddressBytes)
		points := vrutils.DecodePoints(iterator.Value())
		if cb(validatorAddress, points) {
			break
		}
	}
}

func (k Keeper) GetEpochValidatorsPoints(ctx sdk.Context, epoch uint64) []*vrtypes.EpochValidatorPoints {
	var result []*vrtypes.EpochValidatorPoints
	k.IterateValidatorsPoints(ctx, epoch, func(validatorAddress string, points uint64) bool {
		result = append(result, &vrtypes.EpochValidatorPoints{
			ValidatorAddress: validatorAddress,
			EpochPoints:      points,
		})
		return false
	})
	return result
}

func (k Keeper) IterateAllValidatorsPoints(ctx sdk.Context, cb func(epoch uint64, validatorAddress string, points uint64) (stop bool)) {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, vrtypes.KeyPrefixEpochValidatorPoints)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		epoch, validatorAddress, err := vrtypes.ParseEpochValidatorPointsKey(iterator.Key())
		if err != nil {
			ctx.Logger().Error(
				"valrewards: skipping malformed validator points key",
				"key", hex.EncodeToString(iterator.Key()),
				"err", err,
			)
			continue
		}

		points := vrutils.DecodePoints(iterator.Value())
		if cb(epoch, validatorAddress, points) {
			break
		}
	}
}

func (k Keeper) IterateAllValidatorsOutstandingRewards(ctx sdk.Context, cb func(epoch uint64, validatorAddress string, amount sdk.Coin) (stop bool)) {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, vrtypes.KeyPrefixEpochValidatorOutstanding)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		epoch, validatorAddress, err := vrtypes.ParseEpochValidatorOutstandingKey(iterator.Key())
		if err != nil {
			ctx.Logger().Error(
				"valrewards: skipping malformed outstanding rewards key",
				"key", hex.EncodeToString(iterator.Key()),
				"err", err,
			)
			continue
		}

		amount := vrutils.DecodeCoin(iterator.Value())
		if cb(epoch, validatorAddress, amount) {
			break
		}
	}
}

func (k Keeper) GetDelegationRewards(ctx sdk.Context, delegator sdk.AccAddress, epoch uint64) (sdk.Coin, error) {
	if err := k.ensureValidatorOperator(ctx, delegator); err != nil {
		return sdk.Coin{}, err
	}
	validatorAddress := sdk.ValAddress(delegator.Bytes()).String()
	rewards := k.GetValidatorOutstandingReward(ctx, epoch, validatorAddress)
	return rewards, nil
}

func (k Keeper) ClaimOperatorRewards(ctx sdk.Context, operator sdk.AccAddress, epoch uint64) (sdk.Coin, error) {
	return k.ClaimValidatorRewards(ctx, operator, epoch)
}

func (k Keeper) ClaimValidatorRewards(ctx sdk.Context, operator sdk.AccAddress, epoch uint64) (sdk.Coin, error) {
	if err := k.ensureValidatorOperator(ctx, operator); err != nil {
		return sdk.Coin{}, err
	}
	validatorAddress := sdk.ValAddress(operator.Bytes()).String()
	rewards := k.GetValidatorOutstandingReward(ctx, epoch, validatorAddress)

	if rewards.IsZero() {
		return sdk.Coin{}, errors.New(vrtypes.ErrNoOutstandingBalance)
	}

	moduleAddr := k.accountKeeper.GetModuleAddress(vrtypes.ModuleName)
	moduleBalance := k.bankKeeper.GetBalance(ctx, moduleAddr, evmtypes.DefaultEVMDenom)
	if moduleBalance.IsLT(rewards) {
		return sdk.Coin{}, errors.New(vrtypes.ErrInsufficientRewardsBalance)
	}

	k.SetValidatorOutstandingReward(ctx, epoch, validatorAddress, sdk.NewCoin(evmtypes.DefaultEVMDenom, math.ZeroInt()))

	transferCoins := sdk.NewCoins(rewards)
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, vrtypes.ModuleName, operator, transferCoins); err != nil {
		return sdk.Coin{}, err
	}

	return rewards, nil
}

func (k Keeper) ensureValidatorOperator(ctx sdk.Context, operator sdk.AccAddress) error {
	_, err := k.stakingKeeper.GetValidator(ctx, sdk.ValAddress(operator.Bytes()))
	if err != nil {
		return errortypes.ErrUnauthorized.Wrap("target address must be a validator operator")
	}
	return nil
}

func (k Keeper) DepositRewardsPoolCoins(ctx sdk.Context, depositor sdk.AccAddress, amount sdk.Coin) error {
	coins := sdk.NewCoins(amount)
	return k.bankKeeper.SendCoinsFromAccountToModule(ctx, depositor, vrtypes.ModuleName, coins)
}

func (k Keeper) GetRewardsPool(ctx sdk.Context) sdk.Coin {
	moduleAddr := k.accountKeeper.GetModuleAddress(vrtypes.ModuleName)
	return k.bankKeeper.GetBalance(ctx, moduleAddr, evmtypes.DefaultEVMDenom)
}

func (k Keeper) GetValidatorOutstandingRewards(ctx sdk.Context, epoch uint64, validatorAddress string) sdk.Coin {
	return k.GetValidatorOutstandingReward(ctx, epoch, validatorAddress)
}
