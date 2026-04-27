package evmd

import (
	"encoding/json"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/testutil/mock"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/evm/config"
	valrewardstypes "github.com/cosmos/evm/x/valrewards/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

type exportedGenesisFixture struct {
	app               *EVMD
	chainID           string
	delegatorAddr     string
	validatorOperator string
	exported          GenesisState
	consensusParams   *tmproto.ConsensusParams
}

func newExportedGenesisFixture(t *testing.T) exportedGenesisFixture {
	t.Helper()

	resetValRewardsIntegrationRuntime()
	t.Cleanup(resetValRewardsIntegrationRuntime)

	chainID := "cross-genesis-export-test"

	privVal := mock.NewPV()
	pubKey, err := privVal.GetPubKey()
	require.NoError(t, err)

	validator := cmttypes.NewValidator(pubKey, 1)
	valSet := cmttypes.NewValidatorSet([]*cmttypes.Validator{validator})

	delegatorPrivKey := secp256k1.GenPrivKey()
	delegatorAccount := authtypes.NewBaseAccount(
		delegatorPrivKey.PubKey().Address().Bytes(),
		delegatorPrivKey.PubKey(),
		0,
		0,
	)
	delegatorBalance := banktypes.Balance{
		Address: delegatorAccount.GetAddress().String(),
		Coins: sdk.NewCoins(
			sdk.NewCoin(evmtypes.DefaultEVMExtendedDenom, sdkmath.NewInt(1_000_000)),
		),
	}

	app := SetupWithGenesisValSet(
		t,
		chainID,
		config.EVMChainID,
		valSet,
		[]authtypes.GenesisAccount{delegatorAccount},
		delegatorBalance,
	)

	ctx := app.NewContextLegacy(false, tmproto.Header{
		ChainID: chainID,
		Height:  1,
	})

	validators, err := app.StakingKeeper.GetAllValidators(ctx)
	require.NoError(t, err)
	require.Len(t, validators, 1)

	commitForExport(app)

	exportedApp, err := app.ExportAppStateAndValidators(false, nil, nil)
	require.NoError(t, err)

	var exported GenesisState
	require.NoError(t, json.Unmarshal(exportedApp.AppState, &exported))

	return exportedGenesisFixture{
		app:               app,
		chainID:           chainID,
		delegatorAddr:     delegatorAccount.GetAddress().String(),
		validatorOperator: validators[0].GetOperator(),
		exported:          exported,
		consensusParams:   &exportedApp.ConsensusParams,
	}
}

func fixtureValidatorConsAddr(t *testing.T, fixture exportedGenesisFixture) string {
	t.Helper()

	var stakingGen stakingtypes.GenesisState
	fixture.app.AppCodec().MustUnmarshalJSON(fixture.exported[stakingtypes.ModuleName], &stakingGen)
	require.NoError(t, stakingGen.UnpackInterfaces(fixture.app.InterfaceRegistry()))

	for _, validator := range stakingGen.Validators {
		if validator.OperatorAddress != fixture.validatorOperator {
			continue
		}
		consAddr, err := validator.GetConsAddr()
		require.NoError(t, err)
		return sdk.ConsAddress(consAddr).String()
	}

	t.Fatalf("validator %s not found in exported staking genesis", fixture.validatorOperator)
	return ""
}

func TestCrossGenesisValidateAcceptsExportedGenesis(t *testing.T) {
	fixture := newExportedGenesisFixture(t)

	require.NoError(t, CrossGenesisValidate(fixture.app.AppCodec(), fixture.exported))
}

func TestCrossGenesisValidateRejectsMissingDistributionHistoricalRewards(t *testing.T) {
	fixture := newExportedGenesisFixture(t)

	var distrGen distrtypes.GenesisState
	fixture.app.AppCodec().MustUnmarshalJSON(fixture.exported[distrtypes.ModuleName], &distrGen)
	var referencedPeriod uint64
	foundCurrentRewards := false
	for _, record := range distrGen.ValidatorCurrentRewards {
		if record.ValidatorAddress == fixture.validatorOperator {
			referencedPeriod = record.Rewards.Period - 1
			foundCurrentRewards = true
			break
		}
	}
	require.True(t, foundCurrentRewards)

	filtered := make([]distrtypes.ValidatorHistoricalRewardsRecord, 0, len(distrGen.ValidatorHistoricalRewards))
	for _, record := range distrGen.ValidatorHistoricalRewards {
		if record.ValidatorAddress == fixture.validatorOperator && record.Period == referencedPeriod {
			continue
		}
		filtered = append(filtered, record)
	}
	distrGen.ValidatorHistoricalRewards = filtered
	fixture.exported[distrtypes.ModuleName] = fixture.app.AppCodec().MustMarshalJSON(&distrGen)

	err := CrossGenesisValidate(fixture.app.AppCodec(), fixture.exported)
	require.ErrorContains(t, err, "distribution missing historical rewards period")
}

func TestCrossGenesisValidateRejectsMissingSlashingSigningInfo(t *testing.T) {
	fixture := newExportedGenesisFixture(t)

	var slashingGen slashingtypes.GenesisState
	fixture.app.AppCodec().MustUnmarshalJSON(fixture.exported[slashingtypes.ModuleName], &slashingGen)
	slashingGen.SigningInfos = nil
	slashingGen.MissedBlocks = nil
	fixture.exported[slashingtypes.ModuleName] = fixture.app.AppCodec().MustMarshalJSON(&slashingGen)

	err := CrossGenesisValidate(fixture.app.AppCodec(), fixture.exported)
	require.ErrorContains(t, err, "slashing missing signing info")
}

func TestCrossGenesisValidateRejectsMissedBlocksWithoutSigningInfo(t *testing.T) {
	fixture := newExportedGenesisFixture(t)

	var slashingGen slashingtypes.GenesisState
	fixture.app.AppCodec().MustUnmarshalJSON(fixture.exported[slashingtypes.ModuleName], &slashingGen)
	validConsAddr := fixtureValidatorConsAddr(t, fixture)
	filtered := make([]slashingtypes.SigningInfo, 0, len(slashingGen.SigningInfos))
	for _, info := range slashingGen.SigningInfos {
		if info.Address == validConsAddr {
			continue
		}
		filtered = append(filtered, info)
	}
	slashingGen.SigningInfos = filtered
	fixture.exported[slashingtypes.ModuleName] = fixture.app.AppCodec().MustMarshalJSON(&slashingGen)

	err := CrossGenesisValidate(fixture.app.AppCodec(), fixture.exported)
	require.ErrorContains(t, err, "slashing missed_blocks entry")
	require.ErrorContains(t, err, "has no matching signing_info")
}

func TestCrossGenesisValidateRejectsOrphanSlashingSigningInfo(t *testing.T) {
	fixture := newExportedGenesisFixture(t)

	var slashingGen slashingtypes.GenesisState
	fixture.app.AppCodec().MustUnmarshalJSON(fixture.exported[slashingtypes.ModuleName], &slashingGen)
	orphanAddr := sdk.ConsAddress(secp256k1.GenPrivKey().PubKey().Address().Bytes()).String()
	slashingGen.SigningInfos = append(slashingGen.SigningInfos, slashingtypes.SigningInfo{
		Address: orphanAddr,
		ValidatorSigningInfo: slashingtypes.ValidatorSigningInfo{
			Address:     orphanAddr,
			JailedUntil: time.Unix(0, 0).UTC(),
		},
	})
	fixture.exported[slashingtypes.ModuleName] = fixture.app.AppCodec().MustMarshalJSON(&slashingGen)

	err := CrossGenesisValidate(fixture.app.AppCodec(), fixture.exported)
	require.ErrorContains(t, err, "slashing signing_info entry")
	require.ErrorContains(t, err, "has no matching validator")
}

func TestCrossGenesisValidateRejectsOrphanSlashingMissedBlocks(t *testing.T) {
	fixture := newExportedGenesisFixture(t)

	var slashingGen slashingtypes.GenesisState
	fixture.app.AppCodec().MustUnmarshalJSON(fixture.exported[slashingtypes.ModuleName], &slashingGen)
	orphanAddr := sdk.ConsAddress(secp256k1.GenPrivKey().PubKey().Address().Bytes()).String()
	slashingGen.SigningInfos = append(slashingGen.SigningInfos, slashingtypes.SigningInfo{
		Address: orphanAddr,
		ValidatorSigningInfo: slashingtypes.ValidatorSigningInfo{
			Address:     orphanAddr,
			JailedUntil: time.Unix(0, 0).UTC(),
		},
	})
	slashingGen.MissedBlocks = append(slashingGen.MissedBlocks, slashingtypes.ValidatorMissedBlocks{
		Address: orphanAddr,
	})
	fixture.exported[slashingtypes.ModuleName] = fixture.app.AppCodec().MustMarshalJSON(&slashingGen)

	err := CrossGenesisValidate(fixture.app.AppCodec(), fixture.exported)
	require.ErrorContains(t, err, "slashing missed_blocks entry")
	require.ErrorContains(t, err, "has no matching validator")
}

func TestInitChainerRejectsExportedGenesisMissingSlashingSigningInfo(t *testing.T) {
	fixture := newExportedGenesisFixture(t)

	var slashingGen slashingtypes.GenesisState
	fixture.app.AppCodec().MustUnmarshalJSON(fixture.exported[slashingtypes.ModuleName], &slashingGen)
	slashingGen.SigningInfos = nil
	slashingGen.MissedBlocks = nil
	fixture.exported[slashingtypes.ModuleName] = fixture.app.AppCodec().MustMarshalJSON(&slashingGen)

	app, _ := setup(true, 5, fixture.chainID+"-import", config.EVMChainID)
	stateBytes, err := json.MarshalIndent(fixture.exported, "", "  ")
	require.NoError(t, err)

	_, err = app.InitChain(&abci.RequestInitChain{
		Validators:      []abci.ValidatorUpdate{},
		ConsensusParams: fixture.consensusParams,
		AppStateBytes:   stateBytes,
		ChainId:         fixture.chainID + "-import",
	})
	require.ErrorContains(t, err, "slashing missing signing info")
}

func TestCrossGenesisValidateRejectsRedelegationUnknownSourceValidator(t *testing.T) {
	fixture := newExportedGenesisFixture(t)

	var stakingGen stakingtypes.GenesisState
	fixture.app.AppCodec().MustUnmarshalJSON(fixture.exported[stakingtypes.ModuleName], &stakingGen)
	stakingGen.Redelegations = append(stakingGen.Redelegations, stakingtypes.Redelegation{
		DelegatorAddress:    fixture.delegatorAddr,
		ValidatorSrcAddress: sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address().Bytes()).String(),
		ValidatorDstAddress: fixture.validatorOperator,
		Entries: []stakingtypes.RedelegationEntry{
			stakingtypes.NewRedelegationEntry(1, time.Unix(1, 0).UTC(), sdkmath.OneInt(), sdkmath.LegacyOneDec(), 1),
		},
	})
	fixture.exported[stakingtypes.ModuleName] = fixture.app.AppCodec().MustMarshalJSON(&stakingGen)

	err := CrossGenesisValidate(fixture.app.AppCodec(), fixture.exported)
	require.ErrorContains(t, err, "staking redelegation for delegator")
	require.ErrorContains(t, err, "unknown source validator")
}

func TestCrossGenesisValidateRejectsRedelegationUnknownDestinationValidator(t *testing.T) {
	fixture := newExportedGenesisFixture(t)

	var stakingGen stakingtypes.GenesisState
	fixture.app.AppCodec().MustUnmarshalJSON(fixture.exported[stakingtypes.ModuleName], &stakingGen)
	stakingGen.Redelegations = append(stakingGen.Redelegations, stakingtypes.Redelegation{
		DelegatorAddress:    fixture.delegatorAddr,
		ValidatorSrcAddress: fixture.validatorOperator,
		ValidatorDstAddress: sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address().Bytes()).String(),
		Entries: []stakingtypes.RedelegationEntry{
			stakingtypes.NewRedelegationEntry(1, time.Unix(1, 0).UTC(), sdkmath.OneInt(), sdkmath.LegacyOneDec(), 1),
		},
	})
	fixture.exported[stakingtypes.ModuleName] = fixture.app.AppCodec().MustMarshalJSON(&stakingGen)

	err := CrossGenesisValidate(fixture.app.AppCodec(), fixture.exported)
	require.ErrorContains(t, err, "staking redelegation for delegator")
	require.ErrorContains(t, err, "unknown destination validator")
}

func TestCrossGenesisValidateRejectsUnbondingDelegationUnknownValidator(t *testing.T) {
	fixture := newExportedGenesisFixture(t)

	var stakingGen stakingtypes.GenesisState
	fixture.app.AppCodec().MustUnmarshalJSON(fixture.exported[stakingtypes.ModuleName], &stakingGen)
	stakingGen.UnbondingDelegations = append(stakingGen.UnbondingDelegations, stakingtypes.UnbondingDelegation{
		DelegatorAddress: fixture.delegatorAddr,
		ValidatorAddress: sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address().Bytes()).String(),
		Entries: []stakingtypes.UnbondingDelegationEntry{
			stakingtypes.NewUnbondingDelegationEntry(1, time.Unix(1, 0).UTC(), sdkmath.OneInt(), 1),
		},
	})
	fixture.exported[stakingtypes.ModuleName] = fixture.app.AppCodec().MustMarshalJSON(&stakingGen)

	err := CrossGenesisValidate(fixture.app.AppCodec(), fixture.exported)
	require.ErrorContains(t, err, "staking unbonding_delegation for delegator")
	require.ErrorContains(t, err, "unknown validator")
}

func TestCrossGenesisValidateRejectsDistributionCurrentRewardsPeriodZero(t *testing.T) {
	fixture := newExportedGenesisFixture(t)

	var distrGen distrtypes.GenesisState
	fixture.app.AppCodec().MustUnmarshalJSON(fixture.exported[distrtypes.ModuleName], &distrGen)
	for i := range distrGen.ValidatorCurrentRewards {
		if distrGen.ValidatorCurrentRewards[i].ValidatorAddress == fixture.validatorOperator {
			distrGen.ValidatorCurrentRewards[i].Rewards.Period = 0
			break
		}
	}
	fixture.exported[distrtypes.ModuleName] = fixture.app.AppCodec().MustMarshalJSON(&distrGen)

	err := CrossGenesisValidate(fixture.app.AppCodec(), fixture.exported)
	require.ErrorContains(t, err, "distribution current rewards for validator")
	require.ErrorContains(t, err, "invalid period 0")
}

func TestCrossGenesisValidateRejectsDelegatorStartingInfoPreviousPeriodAtOrAboveCurrent(t *testing.T) {
	fixture := newExportedGenesisFixture(t)

	var distrGen distrtypes.GenesisState
	fixture.app.AppCodec().MustUnmarshalJSON(fixture.exported[distrtypes.ModuleName], &distrGen)

	var currentPeriod uint64
	for _, record := range distrGen.ValidatorCurrentRewards {
		if record.ValidatorAddress == fixture.validatorOperator {
			currentPeriod = record.Rewards.Period
			break
		}
	}
	require.NotZero(t, currentPeriod)

	distrGen.DelegatorStartingInfos = append(distrGen.DelegatorStartingInfos, distrtypes.DelegatorStartingInfoRecord{
		DelegatorAddress: fixture.delegatorAddr,
		ValidatorAddress: fixture.validatorOperator,
		StartingInfo:     distrtypes.NewDelegatorStartingInfo(currentPeriod, sdkmath.LegacyOneDec(), 1),
	})
	fixture.exported[distrtypes.ModuleName] = fixture.app.AppCodec().MustMarshalJSON(&distrGen)

	err := CrossGenesisValidate(fixture.app.AppCodec(), fixture.exported)
	require.ErrorContains(t, err, "distribution delegator starting info")
	require.ErrorContains(t, err, "expected less than current period")
}

func TestCrossGenesisValidateRejectsDistributionSlashEventPeriodMismatch(t *testing.T) {
	fixture := newExportedGenesisFixture(t)

	var distrGen distrtypes.GenesisState
	fixture.app.AppCodec().MustUnmarshalJSON(fixture.exported[distrtypes.ModuleName], &distrGen)
	distrGen.ValidatorSlashEvents = append(distrGen.ValidatorSlashEvents, distrtypes.ValidatorSlashEventRecord{
		ValidatorAddress:    fixture.validatorOperator,
		Height:              1,
		Period:              1,
		ValidatorSlashEvent: distrtypes.NewValidatorSlashEvent(2, sdkmath.LegacyOneDec()),
	})
	fixture.exported[distrtypes.ModuleName] = fixture.app.AppCodec().MustMarshalJSON(&distrGen)

	err := CrossGenesisValidate(fixture.app.AppCodec(), fixture.exported)
	require.ErrorContains(t, err, "distribution slash event")
	require.ErrorContains(t, err, "mismatched period")
}

func TestCrossGenesisValidateRejectsOrphanDistributionCurrentRewards(t *testing.T) {
	fixture := newExportedGenesisFixture(t)

	var distrGen distrtypes.GenesisState
	fixture.app.AppCodec().MustUnmarshalJSON(fixture.exported[distrtypes.ModuleName], &distrGen)
	distrGen.ValidatorCurrentRewards = append(distrGen.ValidatorCurrentRewards, distrtypes.ValidatorCurrentRewardsRecord{
		ValidatorAddress: sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address().Bytes()).String(),
		Rewards: distrtypes.ValidatorCurrentRewards{
			Rewards: sdk.DecCoins{},
			Period:  1,
		},
	})
	fixture.exported[distrtypes.ModuleName] = fixture.app.AppCodec().MustMarshalJSON(&distrGen)

	err := CrossGenesisValidate(fixture.app.AppCodec(), fixture.exported)
	require.ErrorContains(t, err, "distribution current_rewards row references unknown validator")
}

func TestCrossGenesisValidateRejectsOrphanDistributionSlashEvent(t *testing.T) {
	fixture := newExportedGenesisFixture(t)

	var distrGen distrtypes.GenesisState
	fixture.app.AppCodec().MustUnmarshalJSON(fixture.exported[distrtypes.ModuleName], &distrGen)
	distrGen.ValidatorSlashEvents = append(distrGen.ValidatorSlashEvents, distrtypes.ValidatorSlashEventRecord{
		ValidatorAddress:    sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address().Bytes()).String(),
		Height:              1,
		Period:              1,
		ValidatorSlashEvent: distrtypes.NewValidatorSlashEvent(1, sdkmath.LegacyOneDec()),
	})
	fixture.exported[distrtypes.ModuleName] = fixture.app.AppCodec().MustMarshalJSON(&distrGen)

	err := CrossGenesisValidate(fixture.app.AppCodec(), fixture.exported)
	require.ErrorContains(t, err, "distribution slash_event references unknown validator")
}

func TestCrossGenesisValidateRejectsGovVoteForMissingProposal(t *testing.T) {
	fixture := newExportedGenesisFixture(t)

	var govGen govv1.GenesisState
	fixture.app.AppCodec().MustUnmarshalJSON(fixture.exported[govtypes.ModuleName], &govGen)
	govGen.Votes = []*govv1.Vote{
		{
			ProposalId: 99,
			Voter:      fixture.delegatorAddr,
		},
	}
	fixture.exported[govtypes.ModuleName] = fixture.app.AppCodec().MustMarshalJSON(&govGen)

	err := CrossGenesisValidate(fixture.app.AppCodec(), fixture.exported)
	require.ErrorContains(t, err, "gov vote references missing proposal 99")
}

func TestCrossGenesisValidateRejectsValRewardsUnknownValidator(t *testing.T) {
	fixture := newExportedGenesisFixture(t)

	var valRewardsGen valrewardstypes.GenesisState
	require.NoError(t, json.Unmarshal(fixture.exported[valrewardstypes.ModuleName], &valRewardsGen))
	valRewardsGen.ValidatorPoints = []valrewardstypes.GenesisValidatorPoint{
		{
			Epoch:            0,
			ValidatorAddress: sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address().Bytes()).String(),
			EpochPoints:      1,
		},
	}
	rawValRewards, err := json.Marshal(&valRewardsGen)
	require.NoError(t, err)
	fixture.exported[valrewardstypes.ModuleName] = rawValRewards

	err = CrossGenesisValidate(fixture.app.AppCodec(), fixture.exported)
	require.ErrorContains(t, err, "valrewards validator_points entry references missing staking validator")
}

func TestCrossGenesisValidateRejectsStaleUpgradePlan(t *testing.T) {
	fixture := newExportedGenesisFixture(t)

	rawUpgradeGenesis, err := json.Marshal(&upgradeGenesisPlanEnvelope{
		Plan: upgradetypes.Plan{
			Name:   "stale-v1",
			Height: 1,
		},
	})
	require.NoError(t, err)
	fixture.exported[upgradetypes.ModuleName] = rawUpgradeGenesis

	err = CrossGenesisValidate(fixture.app.AppCodec(), fixture.exported)
	require.ErrorContains(t, err, `upgrade plan "stale-v1" has stale height 1`)
}

func TestCrossGenesisValidateRejectsStaleUpgradePlanAboveInitialHeight(t *testing.T) {
	fixture := newExportedGenesisFixture(t)

	rawUpgradeGenesis, err := json.Marshal(&upgradeGenesisPlanEnvelope{
		Plan: upgradetypes.Plan{
			Name:   "stale-v100",
			Height: 50,
		},
	})
	require.NoError(t, err)
	fixture.exported[upgradetypes.ModuleName] = rawUpgradeGenesis

	err = CrossGenesisValidateAtInitialHeight(fixture.app.AppCodec(), fixture.exported, 100)
	require.ErrorContains(t, err, `upgrade plan "stale-v100" has stale height 50`)
	require.ErrorContains(t, err, "start height 100")
}

func TestCrossGenesisValidateAcceptsFutureUpgradePlanAboveInitialHeight(t *testing.T) {
	fixture := newExportedGenesisFixture(t)

	rawUpgradeGenesis, err := json.Marshal(&upgradeGenesisPlanEnvelope{
		Plan: upgradetypes.Plan{
			Name:   "future-v101",
			Height: 101,
		},
	})
	require.NoError(t, err)
	fixture.exported[upgradetypes.ModuleName] = rawUpgradeGenesis

	require.NoError(t, CrossGenesisValidateAtInitialHeight(fixture.app.AppCodec(), fixture.exported, 100))
}

func TestInitChainerRejectsStaleUpgradePlanAboveInitialHeight(t *testing.T) {
	fixture := newExportedGenesisFixture(t)

	rawUpgradeGenesis, err := json.Marshal(&upgradeGenesisPlanEnvelope{
		Plan: upgradetypes.Plan{
			Name:   "stale-import",
			Height: 50,
		},
	})
	require.NoError(t, err)
	fixture.exported[upgradetypes.ModuleName] = rawUpgradeGenesis

	appStateBytes, err := json.Marshal(fixture.exported)
	require.NoError(t, err)

	importedApp, _ := setup(false, 5, fixture.chainID+"-stale-import", config.EVMChainID)

	_, err = importedApp.InitChain(&abci.RequestInitChain{
		Validators:      []abci.ValidatorUpdate{},
		ConsensusParams: fixture.consensusParams,
		AppStateBytes:   appStateBytes,
		ChainId:         fixture.chainID + "-stale-import",
		InitialHeight:   100,
	})
	require.ErrorContains(t, err, `upgrade plan "stale-import" has stale height 50`)
	require.ErrorContains(t, err, "start height 100")
}

func TestInitChainerRejectsCrossModuleMismatch(t *testing.T) {
	fixture := newExportedGenesisFixture(t)

	var distrGen distrtypes.GenesisState
	fixture.app.AppCodec().MustUnmarshalJSON(fixture.exported[distrtypes.ModuleName], &distrGen)
	distrGen.ValidatorCurrentRewards = nil
	fixture.exported[distrtypes.ModuleName] = fixture.app.AppCodec().MustMarshalJSON(&distrGen)

	appStateBytes, err := json.Marshal(fixture.exported)
	require.NoError(t, err)

	importedApp, _ := setup(false, 5, fixture.chainID, config.EVMChainID)

	_, err = importedApp.InitChain(&abci.RequestInitChain{
		Validators:      []abci.ValidatorUpdate{},
		ConsensusParams: fixture.consensusParams,
		AppStateBytes:   appStateBytes,
		ChainId:         fixture.chainID,
	})
	require.ErrorContains(t, err, "distribution missing current rewards")
}

// Shorting the bonded_tokens_pool bank balance must be rejected: the sum of
// bonded validator tokens wouldn't match what's actually in the pool, which
// would otherwise panic inside staking.InitGenesis at boot. This check is the
// named, early analogue of that panic.
func TestCrossGenesisValidateRejectsBondedPoolBalanceMismatch(t *testing.T) {
	fixture := newExportedGenesisFixture(t)

	// Resolve the bond denom from the fixture's staking params so the test
	// isn't wedged to a hard-coded "stake" or "ctm".
	var stakingGen stakingtypes.GenesisState
	fixture.app.AppCodec().MustUnmarshalJSON(fixture.exported[stakingtypes.ModuleName], &stakingGen)
	bondDenom := stakingGen.Params.BondDenom

	var bankGen banktypes.GenesisState
	fixture.app.AppCodec().MustUnmarshalJSON(fixture.exported[banktypes.ModuleName], &bankGen)

	bondedPoolAddr := authtypes.NewModuleAddress(stakingtypes.BondedPoolName).String()
	found := false
	for i := range bankGen.Balances {
		if bankGen.Balances[i].Address == bondedPoolAddr {
			bankGen.Balances[i].Coins = bankGen.Balances[i].Coins.Sub(
				sdk.NewCoin(bondDenom, sdkmath.OneInt()),
			)
			found = true
			break
		}
	}
	require.True(t, found, "bonded_tokens_pool must be present in exported bank balances")
	fixture.exported[banktypes.ModuleName] = fixture.app.AppCodec().MustMarshalJSON(&bankGen)

	err := CrossGenesisValidate(fixture.app.AppCodec(), fixture.exported)
	require.ErrorContains(t, err, "bonded_tokens_pool balance")
	require.ErrorContains(t, err, "does not match")
}

// The distribution module-account bank balance must equal
// truncate(community_pool + Σ outstanding_rewards). Drifting the balance by
// +1 stake must be rejected, which is the same class of bug that produced
// the original April 18 symptom when the genesis export omitted
// per-validator rewards state.
func TestCrossGenesisValidateRejectsDistributionModuleBalanceMismatch(t *testing.T) {
	fixture := newExportedGenesisFixture(t)

	var bankGen banktypes.GenesisState
	fixture.app.AppCodec().MustUnmarshalJSON(fixture.exported[banktypes.ModuleName], &bankGen)

	distrAddr := authtypes.NewModuleAddress(distrtypes.ModuleName).String()
	patched := false
	for i := range bankGen.Balances {
		if bankGen.Balances[i].Address == distrAddr {
			// +1 stake drift: any non-zero delta from community_pool + outstanding trips the check.
			bankGen.Balances[i].Coins = bankGen.Balances[i].Coins.Add(
				sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.OneInt()),
			)
			patched = true
			break
		}
	}
	if !patched {
		// The exported fixture may not have a distribution entry yet (empty
		// community_pool + no outstanding rewards). Introduce one and bump
		// supply to keep bank's own invariant intact.
		bankGen.Balances = append(bankGen.Balances, banktypes.Balance{
			Address: distrAddr,
			Coins:   sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.OneInt())),
		})
	}
	if !bankGen.Supply.IsZero() {
		bankGen.Supply = bankGen.Supply.Add(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.OneInt()))
	}
	fixture.exported[banktypes.ModuleName] = fixture.app.AppCodec().MustMarshalJSON(&bankGen)

	err := CrossGenesisValidate(fixture.app.AppCodec(), fixture.exported)
	require.ErrorContains(t, err, "distribution balance")
	require.ErrorContains(t, err, "does not match truncate(community_pool + Σ outstanding_rewards)")
}

// The not_bonded_tokens_pool must equal the sum of non-bonded validator
// tokens + unbonding-delegation entry balances. The fixture has none of
// either, so any non-zero balance there is a mismatch.
func TestCrossGenesisValidateRejectsNotBondedPoolBalanceMismatch(t *testing.T) {
	fixture := newExportedGenesisFixture(t)

	var stakingGen stakingtypes.GenesisState
	fixture.app.AppCodec().MustUnmarshalJSON(fixture.exported[stakingtypes.ModuleName], &stakingGen)
	bondDenom := stakingGen.Params.BondDenom

	var bankGen banktypes.GenesisState
	fixture.app.AppCodec().MustUnmarshalJSON(fixture.exported[banktypes.ModuleName], &bankGen)

	notBondedPoolAddr := authtypes.NewModuleAddress(stakingtypes.NotBondedPoolName).String()
	bankGen.Balances = append(bankGen.Balances, banktypes.Balance{
		Address: notBondedPoolAddr,
		Coins:   sdk.NewCoins(sdk.NewCoin(bondDenom, sdkmath.OneInt())),
	})
	// Keep supply consistent: a non-zero not_bonded entry bumps total supply by 1.
	if !bankGen.Supply.IsZero() {
		bankGen.Supply = bankGen.Supply.Add(sdk.NewCoin(bondDenom, sdkmath.OneInt()))
	}
	fixture.exported[banktypes.ModuleName] = fixture.app.AppCodec().MustMarshalJSON(&bankGen)

	err := CrossGenesisValidate(fixture.app.AppCodec(), fixture.exported)
	require.ErrorContains(t, err, "not_bonded_tokens_pool balance")
	require.ErrorContains(t, err, "does not match")
}
