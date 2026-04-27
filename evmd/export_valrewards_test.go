package evmd

import (
	"bytes"
	"encoding/json"
	"testing"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"

	sdkmath "cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/testutil/mock"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/evm/config"
	"github.com/cosmos/evm/x/valrewards"
	valrewardstypes "github.com/cosmos/evm/x/valrewards/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/stretchr/testify/require"
)

type valRewardsExportFixture struct {
	app               *EVMD
	ctx               sdk.Context
	chainID           string
	depositor         sdk.AccAddress
	validatorOperator string
	operatorAccount   sdk.AccAddress
}

func commitForExport(app *EVMD) {
	app.SimWriteState()
	app.Commit()
}

func requireValRewardsIntegrationRuntime(t *testing.T) {
	t.Helper()

	if !valRewardsIntegrationRuntimeEnabled() {
		t.Skip("requires go test -tags=test for repeatable EVMD export/import integration setup")
	}

	resetValRewardsIntegrationRuntime()
	t.Cleanup(resetValRewardsIntegrationRuntime)
}

func newValRewardsExportFixture(t *testing.T) valRewardsExportFixture {
	t.Helper()

	chainID := "valrewards-export-test"

	privVal := mock.NewPV()
	pubKey, err := privVal.GetPubKey()
	require.NoError(t, err)

	validator := cmttypes.NewValidator(pubKey, 1)
	valSet := cmttypes.NewValidatorSet([]*cmttypes.Validator{validator})

	depositorPrivKey := secp256k1.GenPrivKey()
	depositorAccount := authtypes.NewBaseAccount(
		depositorPrivKey.PubKey().Address().Bytes(),
		depositorPrivKey.PubKey(),
		0,
		0,
	)
	depositorBalance := banktypes.Balance{
		Address: depositorAccount.GetAddress().String(),
		Coins: sdk.NewCoins(
			sdk.NewCoin(evmtypes.DefaultEVMExtendedDenom, sdkmath.NewInt(1_000_000)),
		),
	}

	app := SetupWithGenesisValSet(
		t,
		chainID,
		config.EVMChainID,
		valSet,
		[]authtypes.GenesisAccount{depositorAccount},
		depositorBalance,
	)

	ctx := app.NewContextLegacy(false, tmproto.Header{
		ChainID: chainID,
		Height:  1,
	})

	validators, err := app.StakingKeeper.GetAllValidators(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, validators)

	validatorOperator := validators[0].GetOperator()
	valAddr, err := sdk.ValAddressFromBech32(validatorOperator)
	require.NoError(t, err)

	return valRewardsExportFixture{
		app:               app,
		ctx:               ctx,
		chainID:           chainID,
		depositor:         depositorAccount.GetAddress(),
		validatorOperator: validatorOperator,
		operatorAccount:   sdk.AccAddress(valAddr),
	}
}

func TestValRewardsGenesisRoundTrip(t *testing.T) {
	requireValRewardsIntegrationRuntime(t)

	fixture := newValRewardsExportFixture(t)

	expected := valrewardstypes.GenesisState{
		Params: valrewardstypes.Params{
			Whitelist: []string{fixture.depositor.String()},
		},
		CurrentRewardSettings: valrewardstypes.RewardSettings{
			BlocksInEpoch:   40,
			RewardsPerEpoch: "3000000000000000000",
			RewardingPaused: false,
		},
		NextRewardSettings: valrewardstypes.RewardSettings{
			BlocksInEpoch:   45,
			RewardsPerEpoch: "4000000000000000000",
			RewardingPaused: true,
		},
		EpochState: valrewardstypes.EpochState{
			CurrentEpoch:           7,
			BlocksIntoCurrentEpoch: 3,
		},
		EpochToPay: 7,
		ValidatorPoints: []valrewardstypes.GenesisValidatorPoint{
			{Epoch: 5, ValidatorAddress: fixture.validatorOperator, EpochPoints: 42},
		},
		ValidatorOutstandingRewards: []valrewardstypes.GenesisValidatorOutstandingReward{
			{
				Epoch:            5,
				ValidatorAddress: fixture.validatorOperator,
				Amount:           sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(250)),
			},
		},
	}

	valrewards.InitGenesis(fixture.ctx, fixture.app.ValRewardsKeeper, expected)

	exported := valrewards.ExportGenesis(fixture.ctx, fixture.app.ValRewardsKeeper)
	require.Equal(t, expected, *exported)
}

func TestExportAppStateAndValidatorsIncludesValRewards(t *testing.T) {
	requireValRewardsIntegrationRuntime(t)

	fixture := newValRewardsExportFixture(t)

	rewardCoin := sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(250))
	currentSettings := valrewardstypes.RewardSettings{
		BlocksInEpoch:   40,
		RewardsPerEpoch: "3000000000000000000",
		RewardingPaused: false,
	}
	nextSettings := valrewardstypes.RewardSettings{
		BlocksInEpoch:   45,
		RewardsPerEpoch: "4000000000000000000",
		RewardingPaused: true,
	}

	fixture.app.ValRewardsKeeper.SetParams(fixture.ctx, valrewardstypes.Params{Whitelist: []string{fixture.depositor.String()}})
	fixture.app.ValRewardsKeeper.SetCurrentRewardSettings(fixture.ctx, currentSettings)
	fixture.app.ValRewardsKeeper.SetNextRewardSettings(fixture.ctx, nextSettings)
	fixture.app.ValRewardsKeeper.SetEpochState(fixture.ctx, valrewardstypes.EpochState{CurrentEpoch: 7, BlocksIntoCurrentEpoch: 3})
	fixture.app.ValRewardsKeeper.SetEpochToPay(fixture.ctx, 7)
	fixture.app.ValRewardsKeeper.SetValidatorRewardPoints(fixture.ctx, 5, fixture.validatorOperator, 42)
	fixture.app.ValRewardsKeeper.SetValidatorOutstandingReward(fixture.ctx, 5, fixture.validatorOperator, rewardCoin)
	fixture.app.ValRewardsKeeper.SetValidatorOutstandingReward(
		fixture.ctx,
		6,
		fixture.validatorOperator,
		sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.ZeroInt()),
	)

	require.NoError(t, fixture.app.ValRewardsKeeper.DepositRewardsPoolCoins(fixture.ctx, fixture.depositor, rewardCoin))
	commitForExport(fixture.app)

	exported, err := fixture.app.ExportAppStateAndValidators(false, nil, nil)
	require.NoError(t, err)

	var appState map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(exported.AppState, &appState))

	rawValRewardsState, ok := appState[valrewardstypes.ModuleName]
	require.True(t, ok, "exported app state should include valrewards")
	require.NotEqual(t, "null", string(bytes.TrimSpace(rawValRewardsState)))

	var exportedGenesis valrewardstypes.GenesisState
	require.NoError(t, json.Unmarshal(rawValRewardsState, &exportedGenesis))

	require.Equal(t, valrewardstypes.Params{Whitelist: []string{fixture.depositor.String()}}, exportedGenesis.Params)
	require.Equal(t, currentSettings, exportedGenesis.CurrentRewardSettings)
	require.Equal(t, nextSettings, exportedGenesis.NextRewardSettings)
	require.Equal(t, valrewardstypes.EpochState{CurrentEpoch: 7, BlocksIntoCurrentEpoch: 3}, exportedGenesis.EpochState)
	require.Equal(t, uint64(7), exportedGenesis.EpochToPay)
	require.Equal(t,
		[]valrewardstypes.GenesisValidatorPoint{
			{Epoch: 5, ValidatorAddress: fixture.validatorOperator, EpochPoints: 42},
		},
		exportedGenesis.ValidatorPoints,
	)
	require.Equal(t,
		[]valrewardstypes.GenesisValidatorOutstandingReward{
			{
				Epoch:            5,
				ValidatorAddress: fixture.validatorOperator,
				Amount:           rewardCoin,
			},
		},
		exportedGenesis.ValidatorOutstandingRewards,
	)
}

func TestValRewardsExportImportContinuity(t *testing.T) {
	requireValRewardsIntegrationRuntime(t)

	fixture := newValRewardsExportFixture(t)

	rewardCoin := sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(250))
	currentSettings := valrewardstypes.RewardSettings{
		BlocksInEpoch:   40,
		RewardsPerEpoch: "3000000000000000000",
		RewardingPaused: false,
	}
	nextSettings := valrewardstypes.RewardSettings{
		BlocksInEpoch:   45,
		RewardsPerEpoch: "4000000000000000000",
		RewardingPaused: true,
	}

	fixture.app.ValRewardsKeeper.SetParams(fixture.ctx, valrewardstypes.Params{Whitelist: []string{fixture.depositor.String()}})
	fixture.app.ValRewardsKeeper.SetCurrentRewardSettings(fixture.ctx, currentSettings)
	fixture.app.ValRewardsKeeper.SetNextRewardSettings(fixture.ctx, nextSettings)
	fixture.app.ValRewardsKeeper.SetEpochState(fixture.ctx, valrewardstypes.EpochState{CurrentEpoch: 3, BlocksIntoCurrentEpoch: 5})
	fixture.app.ValRewardsKeeper.SetEpochToPay(fixture.ctx, 3)
	fixture.app.ValRewardsKeeper.SetValidatorRewardPoints(fixture.ctx, 1, fixture.validatorOperator, 10)
	fixture.app.ValRewardsKeeper.SetValidatorRewardPoints(fixture.ctx, 2, fixture.validatorOperator, 15)
	fixture.app.ValRewardsKeeper.SetValidatorOutstandingReward(fixture.ctx, 2, fixture.validatorOperator, rewardCoin)
	require.NoError(t, fixture.app.ValRewardsKeeper.DepositRewardsPoolCoins(fixture.ctx, fixture.depositor, rewardCoin))
	commitForExport(fixture.app)

	exported, err := fixture.app.ExportAppStateAndValidators(false, nil, nil)
	require.NoError(t, err)

	resetValRewardsIntegrationRuntime()
	importedApp, _ := setup(false, 5, fixture.chainID, config.EVMChainID)

	_, err = importedApp.InitChain(&abci.RequestInitChain{
		Validators:      []abci.ValidatorUpdate{},
		ConsensusParams: &exported.ConsensusParams,
		AppStateBytes:   exported.AppState,
		ChainId:         fixture.chainID,
	})
	require.NoError(t, err)

	importedCtx := importedApp.NewContextLegacy(false, tmproto.Header{
		ChainID: fixture.chainID,
		Height:  1,
	})

	exportedGenesis := valrewards.ExportGenesis(importedCtx, importedApp.ValRewardsKeeper)
	require.Equal(t,
		valrewardstypes.GenesisState{
			Params:                valrewardstypes.Params{Whitelist: []string{fixture.depositor.String()}},
			CurrentRewardSettings: currentSettings,
			NextRewardSettings:    nextSettings,
			EpochState:            valrewardstypes.EpochState{CurrentEpoch: 3, BlocksIntoCurrentEpoch: 5},
			EpochToPay:            3,
			ValidatorPoints: []valrewardstypes.GenesisValidatorPoint{
				{Epoch: 1, ValidatorAddress: fixture.validatorOperator, EpochPoints: 10},
				{Epoch: 2, ValidatorAddress: fixture.validatorOperator, EpochPoints: 15},
			},
			ValidatorOutstandingRewards: []valrewardstypes.GenesisValidatorOutstandingReward{
				{
					Epoch:            2,
					ValidatorAddress: fixture.validatorOperator,
					Amount:           rewardCoin,
				},
			},
		},
		*exportedGenesis,
	)

	require.Equal(t, rewardCoin, importedApp.ValRewardsKeeper.GetRewardsPool(importedCtx))

	claimedRewards, err := importedApp.ValRewardsKeeper.ClaimValidatorRewards(importedCtx, fixture.operatorAccount, 2)
	require.NoError(t, err)
	require.Equal(t, rewardCoin, claimedRewards)
	require.True(t, importedApp.ValRewardsKeeper.GetValidatorOutstandingReward(importedCtx, 2, fixture.validatorOperator).IsZero())
}
