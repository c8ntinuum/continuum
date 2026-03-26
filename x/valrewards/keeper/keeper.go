package keeper

import (
	"errors"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	vrtypes "github.com/cosmos/evm/x/valrewards/types"
	vrutils "github.com/cosmos/evm/x/valrewards/utils"
)

type Keeper struct {
	storeKey      storetypes.StoreKey
	stakingKeeper *stakingkeeper.Keeper
	accountKeeper vrtypes.AccountKeeper
	bankKeeper    vrtypes.BankKeeper
}

func NewKeeper(
	storeKey storetypes.StoreKey,
	stakingKeeper *stakingkeeper.Keeper,
	accountKeeper vrtypes.AccountKeeper,
	bankKeeper vrtypes.BankKeeper,
) Keeper {
	return Keeper{
		storeKey:      storeKey,
		stakingKeeper: stakingKeeper,
		accountKeeper: accountKeeper,
		bankKeeper:    bankKeeper,
	}
}

func (k Keeper) BeginBlocker(ctx sdk.Context) error {
	k.processValidatorsPoints(ctx, ctx.BlockHeight())
	k.processValidatorsRewards(ctx, ctx.BlockHeight())
	return nil
}

func (k Keeper) processValidatorsRewards(ctx sdk.Context, currentHeight int64) {

	if currentHeight > 1 {
		prevBlockEpoch := (currentHeight - 1) / vrtypes.BLOCKS_IN_EPOCH
		nextEpochToPay := k.GetEpochToPay(ctx)
		if uint64(prevBlockEpoch) > nextEpochToPay {
			requiredRewards := vrtypes.GetRewardsPerEpoch()
			epochValidatorsToPay := k.GetEpochValidatorsPoints(ctx, nextEpochToPay)
			var epochTotalPoints uint64
			for _, epochValidatorToPay := range epochValidatorsToPay {
				epochTotalPoints += epochValidatorToPay.EpochPoints
			}
			if epochTotalPoints == 0 {
				k.SetEpochToPay(ctx, nextEpochToPay+1)
				return
			}
			for _, epochValidatorToPay := range epochValidatorsToPay {
				paymentFraction := math.LegacyNewDec(int64(epochValidatorToPay.EpochPoints)).QuoTruncate(math.LegacyNewDec(int64(epochTotalPoints)))
				toPayAmount := math.LegacyNewDecFromInt(requiredRewards.Amount).Mul(paymentFraction).TruncateInt()
				toPayCoin := sdk.NewCoin(evmtypes.DefaultEVMDenom, toPayAmount)
				k.SetValidatorOutstandingReward(ctx, nextEpochToPay, epochValidatorToPay.ValidatorAddress, toPayCoin)
			}
			k.SetEpochToPay(ctx, nextEpochToPay+1)
		}
	}
}

func (k Keeper) processValidatorsPoints(ctx sdk.Context, currentHeight int64) {
	if currentHeight > 1 {
		prevBlockEpoch := (currentHeight - 1) / vrtypes.BLOCKS_IN_EPOCH
		var prevBlockTotalPower int64
		for _, voteInfo := range ctx.VoteInfos() {
			prevBlockTotalPower += voteInfo.Validator.Power
		}
		if prevBlockTotalPower == 0 {
			return
		}
		for _, voteInfo := range ctx.VoteInfos() {
			validator, err := k.stakingKeeper.ValidatorByConsAddr(ctx, voteInfo.Validator.Address)
			if err != nil {
				continue
			}
			powerFraction := math.LegacyNewDec(voteInfo.Validator.Power).QuoTruncate(math.LegacyNewDec(prevBlockTotalPower))
			validatorPoints := uint64(powerFraction.MulInt64(100).TruncateInt64())
			k.AddValidatorRewardPoints(ctx, uint64(prevBlockEpoch), validator.GetOperator(), validatorPoints)
			if sdk.ConsAddress(voteInfo.Validator.Address).String() == sdk.ConsAddress(ctx.BlockHeader().ProposerAddress).String() {
				k.AddValidatorRewardPoints(ctx, uint64(prevBlockEpoch), validator.GetOperator(), vrtypes.PROPOSER_BONUS_POINTS)
			}
		}
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

func (k Keeper) GetDelegationRewards(ctx sdk.Context, delegator sdk.AccAddress, maxRetrieve uint32, epoch uint64) (sdk.Coin, error) {
	if err := k.ensureValidatorOperator(ctx, delegator); err != nil {
		return sdk.Coin{}, err
	}
	validatorAddress := sdk.ValAddress(delegator.Bytes()).String()
	rewards := k.GetValidatorOutstandingReward(ctx, epoch, validatorAddress)
	return rewards, nil
}

func (k Keeper) ClaimDelegationRewards(ctx sdk.Context, delegator sdk.AccAddress, maxRetrieve uint32, epoch uint64) (sdk.Coin, error) {
	return k.ClaimValidatorRewards(ctx, delegator, epoch)
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
		return errortypes.ErrUnauthorized.Wrap("only validator operator can claim rewards")
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
