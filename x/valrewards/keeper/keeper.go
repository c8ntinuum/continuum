package keeper

import (
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	sdk "github.com/cosmos/cosmos-sdk/types"

	vrtypes "github.com/cosmos/evm/x/valrewards/types"
	vrutils "github.com/cosmos/evm/x/valrewards/utils"
)

type Keeper struct {
	storeKey      storetypes.StoreKey
	stakingKeeper *stakingkeeper.Keeper
}

func NewKeeper(
	storeKey storetypes.StoreKey,
	stakingKeeper *stakingkeeper.Keeper,
) Keeper {
	return Keeper{
		storeKey:      storeKey,
		stakingKeeper: stakingKeeper,
	}
}

func (k Keeper) BeginBlocker(ctx sdk.Context) error {
	k.processValidatorsPoints(ctx, ctx.BlockHeight())
	k.processValidatorsRewards(ctx, ctx.BlockHeight())
	return nil
}

func (k Keeper) processValidatorsRewards(ctx sdk.Context, currentHeight int64) {

	if currentHeight > 1 {

		// Get previous block epoch
		prevBlockEpoch := (currentHeight - 1) / vrtypes.BLOCKS_IN_EPOCH

		// Get last epoch unsettled
		nextEpochToPay := k.GetEpochToPay(ctx)

		// Check if last block is newer than last unsetlled block
		if uint64(prevBlockEpoch) > nextEpochToPay {

			// Get required rewards per block
			requiredRewards := vrtypes.GetRewardsPerEpoch()

			// Retrieve epoch validators and points
			epochValidatorsToPay := k.GetEpochValidatorsPoints(ctx, nextEpochToPay)

			// Calculate total points in epoch
			var epochTotalPoints uint64
			for _, epochValidatorToPay := range epochValidatorsToPay {
				epochTotalPoints += epochValidatorToPay.EpochPoints
			}

			// Iterate each validator to get each one's amount
			for _, epochValidatorToPay := range epochValidatorsToPay {

				// Calculate validator reward
				paymentFraction := math.LegacyNewDec(int64(epochValidatorToPay.EpochPoints)).QuoTruncate(math.LegacyNewDec(int64(epochTotalPoints)))
				toPayAmount := math.LegacyNewDecFromInt(requiredRewards.Amount).Mul(paymentFraction).TruncateInt()
				toPayCoin := sdk.NewCoin(evmtypes.DefaultEVMDenom, toPayAmount)

				// Register reward
				k.SetValidatorOutstandingReward(ctx, nextEpochToPay, epochValidatorToPay.ValidatorAddress, toPayCoin)
			}

			// Mark epoch as paid
			k.SetEpochToPay(ctx, nextEpochToPay+1)
		}
	}
}

func (k Keeper) processValidatorsPoints(ctx sdk.Context, currentHeight int64) {
	if currentHeight > 1 {

		// Get previous block epoch
		prevBlockEpoch := (currentHeight - 1) / vrtypes.BLOCKS_IN_EPOCH

		// Calculate previous block total power
		var prevBlockTotalPower int64
		for _, voteInfo := range ctx.VoteInfos() {
			prevBlockTotalPower += voteInfo.Validator.Power
		}

		// Iterate block votes
		for _, voteInfo := range ctx.VoteInfos() {

			// Get vote validator by consensus address
			validator, _ := k.stakingKeeper.ValidatorByConsAddr(ctx, voteInfo.Validator.Address)

			// Get voting power fraction for validator
			powerFraction := math.LegacyNewDec(voteInfo.Validator.Power).QuoTruncate(math.LegacyNewDec(prevBlockTotalPower))

			// Calculate validator reward points
			validatorPoints := uint64(powerFraction.MulInt64(100).TruncateInt64())

			// Register points to validator operator address
			k.AddValidatorRewardPoints(ctx, uint64(prevBlockEpoch), validator.GetOperator(), validatorPoints)

			// Check if validator is also proposer
			if sdk.ConsAddress(voteInfo.Validator.Address).String() == sdk.ConsAddress(ctx.BlockHeader().ProposerAddress).String() {

				// Resgiter proposer bonus points as well
				k.AddValidatorRewardPoints(ctx, uint64(prevBlockEpoch), validator.GetOperator(), vrtypes.PROPOSER_BONUS_POINTS)
			}
		}
	}
}

func (k Keeper) GetEpochToPay(ctx sdk.Context) uint64 {
	// Get store
	store := ctx.KVStore(k.storeKey)
	// Get epoch to pay key
	key := vrtypes.GetEpochToPayKey()
	// Get epoch to pay
	bz := store.Get(key)
	// Decode epoch
	return vrutils.DecodeEpochToPay(bz)
}

func (k Keeper) SetEpochToPay(ctx sdk.Context, nextEpoch uint64) {
	// Get store
	store := ctx.KVStore(k.storeKey)
	// Get epoch to pay key
	key := vrtypes.GetEpochToPayKey()
	// Encode epoch
	bz := vrutils.EncodeEpochToPay(nextEpoch)
	// Set next epoch to pay
	store.Set(key, bz)
}

func (k Keeper) SetValidatorOutstandingReward(ctx sdk.Context, epoch uint64, validatorAddress string, amount sdk.Coin) {
	// Get validator (operator) address bytes
	validatorAddressBytes := []byte(validatorAddress)
	// Get store
	store := ctx.KVStore(k.storeKey)
	// Get epoch validator payment key
	key := vrtypes.GetEpochValidatorOutstandingKey(epoch, validatorAddressBytes)
	// Encode amount
	bz := vrutils.EncodeCoin(amount)
	// Set current epoch validator payment
	store.Set(key, bz)
}

func (k Keeper) GetValidatorOutstandingReward(ctx sdk.Context, epoch uint64, validatorAddress string) sdk.Coin {
	// Get validator (operator) address bytes
	validatorAddressBytes := []byte(validatorAddress)
	// Get store
	store := ctx.KVStore(k.storeKey)
	// Get epoch validator payment key
	key := vrtypes.GetEpochValidatorOutstandingKey(epoch, validatorAddressBytes)
	// Get validator payment in epoch
	bz := store.Get(key)
	// Decode amount
	return vrutils.DecodeCoin(bz)
}

func (k Keeper) iterateValidatorsOutstanding(ctx sdk.Context, epoch uint64, cb func(validatorAddress string, amount sdk.Coin) (stop bool)) {
	// Get store
	store := ctx.KVStore(k.storeKey)
	// Get epoch key on top of all validators
	allEpochValidatorsKey := vrtypes.GetEpochValidatorOutstandingListKey(epoch)
	// Define an iterator for the top key
	iterator := storetypes.KVStorePrefixIterator(store, allEpochValidatorsKey)
	defer iterator.Close()
	// Iterate all validtors in epoch
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
	// Get validator (operator) address bytes
	validatorAddressBytes := []byte(validatorAddress)
	// Get store
	store := ctx.KVStore(k.storeKey)
	// Get epoch validator key
	key := vrtypes.GetEpochValidatorPointsKey(epoch, validatorAddressBytes)
	// Get current epoch no of points
	var currentPoints uint64
	if bz := store.Get(key); bz != nil {
		currentPoints = vrutils.DecodePoints(bz)
	}
	// Set current epoch new points
	store.Set(key, vrutils.EncodePoints(currentPoints+points))
}

func (k Keeper) GetValidatorPoints(ctx sdk.Context, epoch uint64, validatorAddress string) uint64 {
	// Get validator (operator) address bytes
	validatorAddressBytes := []byte(validatorAddress)
	// Get store
	store := ctx.KVStore(k.storeKey)
	// Get epoch validator key
	key := vrtypes.GetEpochValidatorPointsKey(epoch, validatorAddressBytes)
	// Get current epoch no of points
	var currentPoints uint64
	if bz := store.Get(key); bz != nil {
		currentPoints = vrutils.DecodePoints(bz)
	}
	return currentPoints
}

func (k Keeper) IterateValidatorsPoints(ctx sdk.Context, epoch uint64, cb func(validatorAddress string, points uint64) (stop bool)) {
	// Get store
	store := ctx.KVStore(k.storeKey)
	// Get epoch key on top of all users
	allEpochValidatorsKey := vrtypes.GetEpochValidatorPointsListKey(epoch)
	// Define an iterator for the top key
	iterator := storetypes.KVStorePrefixIterator(store, allEpochValidatorsKey)
	defer iterator.Close()
	// Iterate all validtors in epoch
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
