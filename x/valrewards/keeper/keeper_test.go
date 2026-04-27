package keeper

import (
	"context"
	"errors"
	"strings"
	"testing"

	"cosmossdk.io/core/address"
	sdkmath "cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
	store "cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	vrtypes "github.com/cosmos/evm/x/valrewards/types"
	vrutils "github.com/cosmos/evm/x/valrewards/utils"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

type validatorFixture struct {
	operatorAcc sdk.AccAddress
	valAddr     sdk.ValAddress
	consAddr    sdk.ConsAddress
}

type fakeStakingKeeper struct {
	maxValidators uint32
	byConsAddr    map[string]stakingtypes.Validator
	byValAddr     map[string]stakingtypes.Validator
}

type fakeAccountKeeper struct {
	moduleAddrs map[string]sdk.AccAddress
}

func (f *fakeAccountKeeper) AddressCodec() address.Codec { return nil }

func (f *fakeAccountKeeper) GetModuleAccount(_ context.Context, _ string) sdk.ModuleAccountI {
	return nil
}

func (f *fakeAccountKeeper) GetModuleAddress(moduleName string) sdk.AccAddress {
	return f.moduleAddrs[moduleName]
}

func (f *fakeAccountKeeper) GetSequence(context.Context, sdk.AccAddress) (uint64, error) {
	return 0, nil
}

type fakeBankKeeper struct {
	balances    map[string]sdk.Coins
	moduleAddrs map[string]sdk.AccAddress
}

func (f *fakeBankKeeper) GetBalance(_ context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	return sdk.NewCoin(denom, f.balances[addr.String()].AmountOf(denom))
}

func (f *fakeBankKeeper) SendCoinsFromAccountToModule(context.Context, sdk.AccAddress, string, sdk.Coins) error {
	return nil
}

func (f *fakeBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	moduleAddr := f.moduleAddrs[senderModule]
	moduleBalance := f.balances[moduleAddr.String()]
	if !moduleBalance.IsAllGTE(amt) {
		return errors.New("insufficient module balance")
	}
	f.balances[moduleAddr.String()] = moduleBalance.Sub(amt...)
	f.balances[recipientAddr.String()] = f.balances[recipientAddr.String()].Add(amt...)
	return nil
}

func (f *fakeStakingKeeper) MaxValidators(_ context.Context) (uint32, error) {
	return f.maxValidators, nil
}

func (f *fakeStakingKeeper) ValidatorByConsAddr(_ context.Context, addr sdk.ConsAddress) (stakingtypes.ValidatorI, error) {
	val, ok := f.byConsAddr[addr.String()]
	if !ok {
		return nil, errors.New("validator not found")
	}
	return val, nil
}

func (f *fakeStakingKeeper) GetValidator(_ context.Context, addr sdk.ValAddress) (stakingtypes.Validator, error) {
	val, ok := f.byValAddr[addr.String()]
	if !ok {
		return stakingtypes.Validator{}, errors.New("validator not found")
	}
	return val, nil
}

func repeatedAddress(seed byte) []byte {
	addr := make([]byte, 20)
	for i := range addr {
		addr[i] = seed
	}
	return addr
}

func setupKeeperWithValidators(t *testing.T, count int) (Keeper, sdk.Context, []validatorFixture) {
	t.Helper()

	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount("cosmos", "cosmospub")
	cfg.SetBech32PrefixForValidator("cosmosvaloper", "cosmosvaloperpub")
	cfg.SetBech32PrefixForConsensusNode("cosmosvalcons", "cosmosvalconspub")

	storeKey := storetypes.NewKVStoreKey(vrtypes.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	ir := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(ir)
	authority := sdk.AccAddress(repeatedAddress(200))
	stakingKeeper := &fakeStakingKeeper{
		maxValidators: uint32(count),
		byConsAddr:    make(map[string]stakingtypes.Validator, count),
		byValAddr:     make(map[string]stakingtypes.Validator, count),
	}

	fixtures := make([]validatorFixture, 0, count)
	for i := 0; i < count; i++ {
		operatorAcc := sdk.AccAddress(repeatedAddress(byte(i + 1)))
		valAddr := sdk.ValAddress(operatorAcc.Bytes())
		consAddr := sdk.ConsAddress(repeatedAddress(byte(i + 101)))
		validator := stakingtypes.Validator{OperatorAddress: valAddr.String()}

		stakingKeeper.byConsAddr[consAddr.String()] = validator
		stakingKeeper.byValAddr[valAddr.String()] = validator
		fixtures = append(fixtures, validatorFixture{
			operatorAcc: operatorAcc,
			valAddr:     valAddr,
			consAddr:    consAddr,
		})
	}

	k := NewKeeper(cdc, storeKey, authority, stakingKeeper, nil, nil)
	return k, ctx, fixtures
}

func setupKeeper(t *testing.T) (Keeper, sdk.Context, sdk.AccAddress, sdk.ValAddress, sdk.ConsAddress) {
	t.Helper()

	k, ctx, fixtures := setupKeeperWithValidators(t, 1)
	return k, ctx, fixtures[0].operatorAcc, fixtures[0].valAddr, fixtures[0].consAddr
}

func storeRawRewardSettings(t *testing.T, k Keeper, ctx sdk.Context, key []byte, settings vrtypes.RewardSettings) {
	t.Helper()

	ctx.KVStore(k.storeKey).Set(key, k.cdc.MustMarshal(&settings))
}

func TestParamsRoundTrip(t *testing.T) {
	k, ctx, operatorAcc, _, _ := setupKeeper(t)
	params := vrtypes.Params{Whitelist: []string{operatorAcc.String()}}

	require.NoError(t, params.Validate())
	k.SetParams(ctx, params)
	require.Equal(t, params, k.GetParams(ctx))
	require.True(t, k.IsWhitelisted(ctx, operatorAcc))
}

func TestUpdateParamsAuthorityOnly(t *testing.T) {
	k, ctx, operatorAcc, _, _ := setupKeeper(t)

	_, err := k.UpdateParams(sdk.WrapSDKContext(ctx), &vrtypes.MsgUpdateParams{
		Authority: k.authority.String(),
		Params:    &vrtypes.Params{Whitelist: []string{operatorAcc.String()}},
	})
	require.NoError(t, err)
	require.Equal(t, []string{operatorAcc.String()}, k.GetParams(ctx).Whitelist)

	_, err = k.UpdateParams(sdk.WrapSDKContext(ctx), &vrtypes.MsgUpdateParams{
		Authority: sdk.AccAddress([]byte("bad_authority")).String(),
		Params:    &vrtypes.Params{Whitelist: []string{operatorAcc.String()}},
	})
	require.Error(t, err)

	canonicalAuthority := k.authority.String()
	upperAuthority := strings.ToUpper(canonicalAuthority)
	if upperAuthority != canonicalAuthority {
		_, err = k.UpdateParams(sdk.WrapSDKContext(ctx), &vrtypes.MsgUpdateParams{
			Authority: upperAuthority,
			Params:    &vrtypes.Params{Whitelist: []string{operatorAcc.String()}},
		})
		require.NoError(t, err)
	}
}

func TestParamsRejectsNilRequest(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	_, err := k.Params(sdk.WrapSDKContext(ctx), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty request")
}

func TestRewardsPoolRejectsNilRequest(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	_, err := k.RewardsPool(sdk.WrapSDKContext(ctx), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty request")
}

func TestValidatorOutstandingRewardsRejectsOverlongAddress(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	_, err := k.ValidatorOutstandingRewards(sdk.WrapSDKContext(ctx), &vrtypes.QueryValidatorOutstandingRewardsRequest{
		Epoch:            0,
		ValidatorAddress: strings.Repeat("a", 129),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "validator address exceeds maximum length")
	require.NotContains(t, err.Error(), strings.Repeat("a", 129))
}

func TestValidatorOutstandingRewardsRejectsHexAddress(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	_, err := k.ValidatorOutstandingRewards(sdk.WrapSDKContext(ctx), &vrtypes.QueryValidatorOutstandingRewardsRequest{
		Epoch:            0,
		ValidatorAddress: "0x0000000000000000000000000000000000000001",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid validator address")
}

func TestSetBlocksInEpochStagesNextEpochSettingsOnly(t *testing.T) {
	k, ctx, operatorAcc, _, _ := setupKeeper(t)
	k.SetParams(ctx, vrtypes.Params{Whitelist: []string{operatorAcc.String()}})
	k.SetCurrentRewardSettings(ctx, vrtypes.DefaultRewardSettings())
	k.SetNextRewardSettings(ctx, vrtypes.DefaultRewardSettings())

	_, err := k.SetBlocksInEpoch(sdk.WrapSDKContext(ctx), &vrtypes.MsgSetBlocksInEpoch{
		Signer:        operatorAcc.String(),
		BlocksInEpoch: 25,
	})
	require.NoError(t, err)
	require.Equal(t, vrtypes.BLOCKS_IN_EPOCH, k.GetCurrentRewardSettings(ctx).BlocksInEpoch)
	require.Equal(t, int64(25), k.GetNextRewardSettings(ctx).BlocksInEpoch)

	_, err = k.SetBlocksInEpoch(sdk.WrapSDKContext(ctx), &vrtypes.MsgSetBlocksInEpoch{
		Signer:        sdk.AccAddress([]byte("not_whitelisted")).String(),
		BlocksInEpoch: 30,
	})
	require.Error(t, err)
}

func TestSetRewardsPerEpochStagesNextEpochSettingsOnly(t *testing.T) {
	k, ctx, operatorAcc, _, _ := setupKeeper(t)
	k.SetParams(ctx, vrtypes.Params{Whitelist: []string{operatorAcc.String()}})
	k.SetCurrentRewardSettings(ctx, vrtypes.DefaultRewardSettings())
	k.SetNextRewardSettings(ctx, vrtypes.DefaultRewardSettings())

	_, err := k.SetRewardsPerEpoch(sdk.WrapSDKContext(ctx), &vrtypes.MsgSetRewardsPerEpoch{
		Signer:          operatorAcc.String(),
		RewardsPerEpoch: "2000000000000000000",
	})
	require.NoError(t, err)
	require.Equal(t, vrtypes.REWARDS_PER_EPOCH, k.GetCurrentRewardSettings(ctx).RewardsPerEpoch)
	require.Equal(t, "2000000000000000000", k.GetNextRewardSettings(ctx).RewardsPerEpoch)

	_, err = k.SetRewardsPerEpoch(sdk.WrapSDKContext(ctx), &vrtypes.MsgSetRewardsPerEpoch{
		Signer:          sdk.AccAddress([]byte("not_whitelisted")).String(),
		RewardsPerEpoch: "3000000000000000000",
	})
	require.Error(t, err)
}

func TestSetRewardingPausedStagesNextEpochSettingsOnly(t *testing.T) {
	k, ctx, operatorAcc, _, _ := setupKeeper(t)
	k.SetParams(ctx, vrtypes.Params{Whitelist: []string{operatorAcc.String()}})
	k.SetCurrentRewardSettings(ctx, vrtypes.DefaultRewardSettings())
	k.SetNextRewardSettings(ctx, vrtypes.DefaultRewardSettings())

	_, err := k.SetRewardingPaused(sdk.WrapSDKContext(ctx), &vrtypes.MsgSetRewardingPaused{
		Signer:          operatorAcc.String(),
		RewardingPaused: true,
	})
	require.NoError(t, err)
	require.False(t, k.GetCurrentRewardSettings(ctx).RewardingPaused)
	require.True(t, k.GetNextRewardSettings(ctx).RewardingPaused)

	_, err = k.SetRewardingPaused(sdk.WrapSDKContext(ctx), &vrtypes.MsgSetRewardingPaused{
		Signer:          sdk.AccAddress([]byte("not_whitelisted")).String(),
		RewardingPaused: false,
	})
	require.Error(t, err)
}

func TestClaimRewardsAllowsRequesterAndPaysValidatorOperator(t *testing.T) {
	k, ctx, operatorAcc, valAddr, _ := setupKeeper(t)
	requester := sdk.AccAddress(repeatedAddress(77))
	moduleAddr := sdk.AccAddress(repeatedAddress(88))
	reward := sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(25))

	k.accountKeeper = &fakeAccountKeeper{
		moduleAddrs: map[string]sdk.AccAddress{
			vrtypes.ModuleName: moduleAddr,
		},
	}
	k.bankKeeper = &fakeBankKeeper{
		balances: map[string]sdk.Coins{
			moduleAddr.String(): sdk.NewCoins(reward),
		},
		moduleAddrs: map[string]sdk.AccAddress{
			vrtypes.ModuleName: moduleAddr,
		},
	}
	k.SetValidatorOutstandingReward(ctx, 3, valAddr.String(), reward)

	_, err := k.ClaimRewards(sdk.WrapSDKContext(ctx), &vrtypes.MsgClaimRewards{
		ValidatorOperator: operatorAcc.String(),
		Epoch:             3,
		Requester:         requester.String(),
	})
	require.NoError(t, err)

	require.True(t, k.GetValidatorOutstandingReward(ctx, 3, valAddr.String()).IsZero())
	require.Equal(t, reward, k.bankKeeper.GetBalance(ctx, operatorAcc, evmtypes.DefaultEVMDenom))
	require.True(t, k.bankKeeper.GetBalance(ctx, requester, evmtypes.DefaultEVMDenom).IsZero())
}

func TestClaimRewardsRejectsNonValidatorTargetEvenWithRequester(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)
	requester := sdk.AccAddress(repeatedAddress(77))
	nonValidator := sdk.AccAddress(repeatedAddress(99))

	_, err := k.ClaimRewards(sdk.WrapSDKContext(ctx), &vrtypes.MsgClaimRewards{
		ValidatorOperator: nonValidator.String(),
		Epoch:             3,
		Requester:         requester.String(),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "target address must be a validator operator")
}

func TestBeginBlockerActivatesPendingSettingsAtEpochBoundary(t *testing.T) {
	k, ctx, _, valAddr, consAddr := setupKeeper(t)

	currentSettings := vrtypes.RewardSettings{
		BlocksInEpoch:   20,
		RewardsPerEpoch: vrtypes.MinRewardsPerEpoch,
		RewardingPaused: false,
	}
	nextSettings := vrtypes.RewardSettings{
		BlocksInEpoch:   25,
		RewardsPerEpoch: "2000000000000000000",
		RewardingPaused: true,
	}

	k.SetCurrentRewardSettings(ctx, currentSettings)
	k.SetNextRewardSettings(ctx, nextSettings)
	k.SetEpochState(ctx, vrtypes.DefaultEpochState())
	k.SetEpochToPay(ctx, 0)

	voteInfos := []abci.VoteInfo{{
		Validator: abci.Validator{
			Address: consAddr.Bytes(),
			Power:   100,
		},
	}}

	for height := int64(2); height <= 21; height++ {
		ctx = ctx.WithBlockHeader(cmtproto.Header{
			Height:          height,
			ProposerAddress: consAddr.Bytes(),
		}).WithVoteInfos(voteInfos)
		require.NoError(t, k.BeginBlocker(ctx))
	}

	state := k.GetEpochState(ctx)
	require.Equal(t, uint64(1), state.CurrentEpoch)
	require.Zero(t, state.BlocksIntoCurrentEpoch)
	require.Equal(t, nextSettings, k.GetCurrentRewardSettings(ctx))
	require.Equal(t, nextSettings, k.GetNextRewardSettings(ctx))
	require.Equal(t, uint64(1), k.GetEpochToPay(ctx))

	reward := k.GetValidatorOutstandingReward(ctx, 0, valAddr.String())
	require.Equal(t, evmtypes.DefaultEVMDenom, reward.Denom)
	require.Equal(t, vrtypes.MinRewardsPerEpoch, reward.Amount.String())
	require.Equal(t, uint64(20*101), k.GetValidatorPoints(ctx, 0, valAddr.String()))

	ctx = ctx.WithBlockHeader(cmtproto.Header{
		Height:          22,
		ProposerAddress: consAddr.Bytes(),
	}).WithVoteInfos(voteInfos)
	require.NoError(t, k.BeginBlocker(ctx))

	state = k.GetEpochState(ctx)
	require.Equal(t, uint64(1), state.CurrentEpoch)
	require.Equal(t, int64(1), state.BlocksIntoCurrentEpoch)
	require.Zero(t, k.GetValidatorPoints(ctx, 1, valAddr.String()))
}

func TestBeginBlockerSplitsRewardsAcrossMultipleValidators(t *testing.T) {
	k, ctx, fixtures := setupKeeperWithValidators(t, 2)

	currentSettings := vrtypes.RewardSettings{
		BlocksInEpoch:   20,
		RewardsPerEpoch: "2020000000000000000000",
		RewardingPaused: false,
	}
	k.SetCurrentRewardSettings(ctx, currentSettings)
	k.SetNextRewardSettings(ctx, currentSettings)
	k.SetEpochState(ctx, vrtypes.DefaultEpochState())
	k.SetEpochToPay(ctx, 0)

	voteInfos := []abci.VoteInfo{
		{
			Validator: abci.Validator{
				Address: fixtures[0].consAddr.Bytes(),
				Power:   60,
			},
		},
		{
			Validator: abci.Validator{
				Address: fixtures[1].consAddr.Bytes(),
				Power:   40,
			},
		},
	}

	for height := int64(2); height <= 21; height++ {
		proposerAddr := fixtures[(height-2)%2].consAddr
		ctx = ctx.WithBlockHeader(cmtproto.Header{
			Height:          height,
			ProposerAddress: proposerAddr.Bytes(),
		}).WithVoteInfos(voteInfos)
		require.NoError(t, k.BeginBlocker(ctx))
	}

	require.Equal(t, uint64(1210), k.GetValidatorPoints(ctx, 0, fixtures[0].valAddr.String()))
	require.Equal(t, uint64(810), k.GetValidatorPoints(ctx, 0, fixtures[1].valAddr.String()))

	rewardOne := k.GetValidatorOutstandingReward(ctx, 0, fixtures[0].valAddr.String())
	rewardTwo := k.GetValidatorOutstandingReward(ctx, 0, fixtures[1].valAddr.String())
	totalRewards, err := vrtypes.ParseRewardsPerEpoch(currentSettings.RewardsPerEpoch)
	require.NoError(t, err)
	totalPoints := sdkmath.LegacyNewDec(2020)
	expectedRewardOne := sdk.NewCoin(
		evmtypes.DefaultEVMDenom,
		sdkmath.LegacyNewDecFromInt(totalRewards).
			Mul(sdkmath.LegacyNewDec(1210).QuoTruncate(totalPoints)).
			TruncateInt(),
	)
	expectedRewardTwo := sdk.NewCoin(
		evmtypes.DefaultEVMDenom,
		sdkmath.LegacyNewDecFromInt(totalRewards).
			Mul(sdkmath.LegacyNewDec(810).QuoTruncate(totalPoints)).
			TruncateInt(),
	)

	require.Equal(t, expectedRewardOne, rewardOne)
	require.Equal(t, expectedRewardTwo, rewardTwo)
	require.True(t, rewardOne.Amount.Add(rewardTwo.Amount).LTE(totalRewards))

	state := k.GetEpochState(ctx)
	require.Equal(t, uint64(1), state.CurrentEpoch)
	require.Zero(t, state.BlocksIntoCurrentEpoch)
	require.Equal(t, uint64(1), k.GetEpochToPay(ctx))
}

func TestRewardSettingsSettersRejectInvalidSettings(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)
	invalid := vrtypes.RewardSettings{
		BlocksInEpoch:   vrtypes.MinBlocksInEpoch,
		RewardsPerEpoch: "",
		RewardingPaused: false,
	}

	require.Panics(t, func() {
		k.SetCurrentRewardSettings(ctx, invalid)
	})
	require.Panics(t, func() {
		k.SetNextRewardSettings(ctx, invalid)
	})
}

func TestBeginBlockerSkipsInvalidCurrentRewardSettings(t *testing.T) {
	k, ctx, _, _, consAddr := setupKeeper(t)

	currentSettings := vrtypes.RewardSettings{
		BlocksInEpoch:   20,
		RewardsPerEpoch: "",
		RewardingPaused: false,
	}
	nextSettings := vrtypes.RewardSettings{
		BlocksInEpoch:   25,
		RewardsPerEpoch: "2000000000000000000",
		RewardingPaused: false,
	}

	storeRawRewardSettings(t, k, ctx, vrtypes.GetCurrentRewardSettingsKey(), currentSettings)
	k.SetNextRewardSettings(ctx, nextSettings)
	k.SetEpochState(ctx, vrtypes.DefaultEpochState())
	k.SetEpochToPay(ctx, 0)

	voteInfos := []abci.VoteInfo{{
		Validator: abci.Validator{
			Address: consAddr.Bytes(),
			Power:   100,
		},
	}}

	for height := int64(2); height <= 21; height++ {
		ctx = ctx.WithBlockHeader(cmtproto.Header{
			Height:          height,
			ProposerAddress: consAddr.Bytes(),
		}).WithVoteInfos(voteInfos)
		require.NoError(t, k.BeginBlocker(ctx))
	}

	state := k.GetEpochState(ctx)
	require.Equal(t, uint64(1), state.CurrentEpoch)
	require.Zero(t, state.BlocksIntoCurrentEpoch)
	require.Equal(t, nextSettings, k.GetCurrentRewardSettings(ctx))
	require.Equal(t, nextSettings, k.GetNextRewardSettings(ctx))
	require.Equal(t, uint64(1), k.GetEpochToPay(ctx))
}

func TestBeginBlockerFallsBackWhenNextRewardSettingsAreInvalid(t *testing.T) {
	k, ctx, _, valAddr, consAddr := setupKeeper(t)

	currentSettings := vrtypes.RewardSettings{
		BlocksInEpoch:   20,
		RewardsPerEpoch: vrtypes.MinRewardsPerEpoch,
		RewardingPaused: false,
	}
	invalidNextSettings := vrtypes.RewardSettings{
		BlocksInEpoch:   25,
		RewardsPerEpoch: "",
		RewardingPaused: false,
	}

	k.SetCurrentRewardSettings(ctx, currentSettings)
	storeRawRewardSettings(t, k, ctx, vrtypes.GetNextRewardSettingsKey(), invalidNextSettings)
	k.SetEpochState(ctx, vrtypes.DefaultEpochState())
	k.SetEpochToPay(ctx, 0)

	voteInfos := []abci.VoteInfo{{
		Validator: abci.Validator{
			Address: consAddr.Bytes(),
			Power:   100,
		},
	}}

	for height := int64(2); height <= 21; height++ {
		ctx = ctx.WithBlockHeader(cmtproto.Header{
			Height:          height,
			ProposerAddress: consAddr.Bytes(),
		}).WithVoteInfos(voteInfos)
		require.NoError(t, k.BeginBlocker(ctx))
	}

	state := k.GetEpochState(ctx)
	require.Equal(t, uint64(1), state.CurrentEpoch)
	require.Zero(t, state.BlocksIntoCurrentEpoch)
	require.Equal(t, currentSettings, k.GetCurrentRewardSettings(ctx))
	require.Equal(t, currentSettings, k.GetNextRewardSettings(ctx))
	require.Equal(t, uint64(1), k.GetEpochToPay(ctx))

	reward := k.GetValidatorOutstandingReward(ctx, 0, valAddr.String())
	require.Equal(t, vrtypes.MinRewardsPerEpoch, reward.Amount.String())
}

func TestIterateAllValidatorsPointsSkipsMalformedKeys(t *testing.T) {
	k, ctx, _, valAddr, _ := setupKeeper(t)

	k.SetValidatorRewardPoints(ctx, 7, valAddr.String(), 42)
	ctx.KVStore(k.storeKey).Set(vrtypes.KeyPrefixEpochValidatorPoints, vrutils.EncodePoints(9))

	var visited int
	require.NotPanics(t, func() {
		k.IterateAllValidatorsPoints(ctx, func(epoch uint64, validatorAddress string, points uint64) bool {
			visited++
			require.Equal(t, uint64(7), epoch)
			require.Equal(t, valAddr.String(), validatorAddress)
			require.Equal(t, uint64(42), points)
			return false
		})
	})
	require.Equal(t, 1, visited)
}

func TestIterateAllValidatorsOutstandingRewardsSkipsMalformedKeys(t *testing.T) {
	k, ctx, _, valAddr, _ := setupKeeper(t)

	reward := sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(99))
	k.SetValidatorOutstandingReward(ctx, 3, valAddr.String(), reward)
	ctx.KVStore(k.storeKey).Set(vrtypes.KeyPrefixEpochValidatorOutstanding, vrutils.EncodeCoin(reward))

	var visited int
	require.NotPanics(t, func() {
		k.IterateAllValidatorsOutstandingRewards(ctx, func(epoch uint64, validatorAddress string, amount sdk.Coin) bool {
			visited++
			require.Equal(t, uint64(3), epoch)
			require.Equal(t, valAddr.String(), validatorAddress)
			require.Equal(t, reward, amount)
			return false
		})
	})
	require.Equal(t, 1, visited)
}
