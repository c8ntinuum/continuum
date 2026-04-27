package evmd

import (
	"encoding/json"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/testutil/mock"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	slashing "github.com/cosmos/cosmos-sdk/x/slashing"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/evm/config"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

// slashingFixture is a fully initialized EVMD app with N bonded genesis validators.
// Callers pass a mutator for the slashing genesis to simulate healthy or broken states.
type slashingFixture struct {
	app         *EVMD
	chainID     string
	validators  []*cmttypes.Validator
	consAddrs   []sdk.ConsAddress
	slashParams slashingtypes.Params
}

// newSlashingFixture builds an EVMD with numValidators bonded genesis validators
// and custom slashing params tuned for fast downtime tests. If mutateSlashing is
// non-nil it is applied to the slashing genesis before InitChain, so tests can
// simulate a broken export (e.g. nil SigningInfos).
func newSlashingFixture(
	t *testing.T,
	chainID string,
	numValidators int,
	slashParams slashingtypes.Params,
	mutateSlashing func(*slashingtypes.GenesisState, []sdk.ConsAddress),
) slashingFixture {
	t.Helper()

	// Reset VM singleton config so InitChain can re-seal it in this test.
	// Only has an effect under the `test` build tag, which is how this package
	// expects multi-InitChain tests to be run.
	resetValRewardsIntegrationRuntime()
	t.Cleanup(resetValRewardsIntegrationRuntime)

	vals := make([]*cmttypes.Validator, 0, numValidators)
	for i := 0; i < numValidators; i++ {
		pv := mock.NewPV()
		pk, err := pv.GetPubKey()
		require.NoError(t, err)
		vals = append(vals, cmttypes.NewValidator(pk, 1))
	}
	valSet := cmttypes.NewValidatorSet(vals)
	consAddrs := make([]sdk.ConsAddress, 0, numValidators)
	for _, v := range vals {
		consAddrs = append(consAddrs, sdk.ConsAddress(v.Address.Bytes()))
	}

	depositorKey := secp256k1.GenPrivKey()
	acc := authtypes.NewBaseAccount(depositorKey.PubKey().Address().Bytes(), depositorKey.PubKey(), 0, 0)
	balance := banktypes.Balance{
		Address: acc.GetAddress().String(),
		Coins:   sdk.NewCoins(sdk.NewCoin(evmtypes.DefaultEVMExtendedDenom, sdkmath.NewInt(1_000_000_000))),
	}

	app, genesisState := setup(true, 5, chainID, config.EVMChainID)
	genesisState, err := simtestutil.GenesisStateWithValSet(app.AppCodec(), genesisState, valSet, []authtypes.GenesisAccount{acc}, balance)
	require.NoError(t, err)

	var bankGenesis banktypes.GenesisState
	app.AppCodec().MustUnmarshalJSON(genesisState[banktypes.ModuleName], &bankGenesis)
	bankGenesis.DenomMetadata = []banktypes.Metadata{defaultDenomMetadata(evmtypes.DefaultEVMExtendedDenom)}
	// GenesisStateWithValSet only credits one bondAmt to the bonded pool even when
	// numValidators > 1, while it bumps total supply by numValidators * bondAmt —
	// bank InitGenesis then panics on the mismatch. Rewrite the bonded-pool balance
	// to match the true bonded stake.
	bondedPoolAddr := authtypes.NewModuleAddress(stakingtypes.BondedPoolName).String()
	totalBonded := sdk.DefaultPowerReduction.MulRaw(int64(numValidators))
	for i := range bankGenesis.Balances {
		if bankGenesis.Balances[i].Address == bondedPoolAddr {
			bankGenesis.Balances[i].Coins = sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, totalBonded))
			break
		}
	}
	genesisState[banktypes.ModuleName] = app.AppCodec().MustMarshalJSON(&bankGenesis)

	var slashingGen slashingtypes.GenesisState
	app.AppCodec().MustUnmarshalJSON(genesisState[slashingtypes.ModuleName], &slashingGen)
	slashingGen.Params = slashParams
	if mutateSlashing != nil {
		mutateSlashing(&slashingGen, consAddrs)
	}
	genesisState[slashingtypes.ModuleName] = app.AppCodec().MustMarshalJSON(&slashingGen)

	stateBytes, err := json.MarshalIndent(genesisState, "", " ")
	require.NoError(t, err)

	_, err = app.InitChain(&abci.RequestInitChain{
		Validators:      []abci.ValidatorUpdate{},
		ConsensusParams: simtestutil.DefaultConsensusParams,
		AppStateBytes:   stateBytes,
		ChainId:         chainID,
	})
	require.NoError(t, err)

	return slashingFixture{
		app:         app,
		chainID:     chainID,
		validators:  vals,
		consAddrs:   consAddrs,
		slashParams: slashParams,
	}
}

// fastDowntimeParams returns slashing params tuned so downtime jailing fires
// inside a short test loop: 10-block window, 50% threshold (maxMissed=5).
func fastDowntimeParams() slashingtypes.Params {
	return slashingtypes.Params{
		SignedBlocksWindow:      10,
		MinSignedPerWindow:      sdkmath.LegacyNewDecWithPrec(5, 1),
		DowntimeJailDuration:    10 * time.Minute,
		SlashFractionDoubleSign: sdkmath.LegacyNewDecWithPrec(5, 2),
		SlashFractionDowntime:   sdkmath.LegacyNewDecWithPrec(1, 2),
	}
}

// simulateAbsent advances block height and runs the slashing BeginBlocker with
// vote infos for every bonded validator. The target validator is marked absent,
// all others are marked as having signed the block.
func simulateAbsent(
	t *testing.T,
	f slashingFixture,
	consAddr sdk.ConsAddress,
	blocks int,
) {
	t.Helper()
	for h := int64(1); h <= int64(blocks); h++ {
		ctx := f.ctxFactory()(h)
		voteInfos := make([]abci.VoteInfo, 0, len(f.validators))
		for _, validator := range f.validators {
			flag := tmproto.BlockIDFlagCommit
			if string(validator.Address.Bytes()) == string(consAddr) {
				flag = tmproto.BlockIDFlagAbsent
			}
			voteInfos = append(voteInfos, abci.VoteInfo{
				Validator: abci.Validator{
					Address: validator.Address.Bytes(),
					Power:   validator.VotingPower,
				},
				BlockIdFlag: flag,
			})
		}
		err := slashing.BeginBlocker(ctx.WithVoteInfos(voteInfos), f.app.SlashingKeeper)
		require.NoErrorf(t, err, "slashing BeginBlocker errored at height %d (validator %s) — pre-fix panic regression", h, consAddr)
	}
}

type slashingfixtureCtxFactory func(height int64) sdk.Context

func (f slashingFixture) ctxFactory() slashingfixtureCtxFactory {
	return func(height int64) sdk.Context {
		return f.app.NewContextLegacy(false, tmproto.Header{
			ChainID: f.chainID,
			Height:  height,
			Time:    time.Unix(0, 0).UTC(),
		})
	}
}

// --------------------------------------------------------------------------
// Tests
// --------------------------------------------------------------------------

// Multi-validator genesis with SigningInfos/MissedBlocks dropped must be
// transparently seeded by the InitChainer for every validator.
func TestSlashingRegression_SeedSigningInfos_MultipleValidators(t *testing.T) {
	f := newSlashingFixture(t, "slashing-seed-multi", 3, slashingtypes.DefaultParams(), func(g *slashingtypes.GenesisState, _ []sdk.ConsAddress) {
		g.SigningInfos = nil
		g.MissedBlocks = nil
	})

	ctx := f.ctxFactory()(1)
	for i, consAddr := range f.consAddrs {
		info, err := f.app.SlashingKeeper.GetValidatorSigningInfo(ctx, consAddr)
		require.NoErrorf(t, err, "validator %d missing signing info after seed", i)
		require.Zero(t, info.IndexOffset)
		require.Zero(t, info.MissedBlocksCounter)
		require.Equal(t, int64(0), info.StartHeight)
	}
}

// If one validator already has signing info with non-zero state in genesis, the
// InitChainer seed logic must not clobber it — only fill gaps.
func TestSlashingRegression_SeedSigningInfos_PreservesExisting(t *testing.T) {
	var preservedInfo slashingtypes.ValidatorSigningInfo

	f := newSlashingFixture(t, "slashing-seed-preserve", 3, slashingtypes.DefaultParams(), func(g *slashingtypes.GenesisState, consAddrs []sdk.ConsAddress) {
		preservedInfo = slashingtypes.NewValidatorSigningInfo(
			consAddrs[0],
			42,
			7,
			time.Unix(0, 0),
			false,
			3,
		)
		g.SigningInfos = []slashingtypes.SigningInfo{{
			Address:              consAddrs[0].String(),
			ValidatorSigningInfo: preservedInfo,
		}}
		g.MissedBlocks = []slashingtypes.ValidatorMissedBlocks{{
			Address: consAddrs[0].String(),
		}}
	})

	ctx := f.ctxFactory()(1)

	got, err := f.app.SlashingKeeper.GetValidatorSigningInfo(ctx, f.consAddrs[0])
	require.NoError(t, err)
	require.Equal(t, int64(42), got.StartHeight, "seed overwrote existing StartHeight")
	require.Equal(t, int64(7), got.IndexOffset, "seed overwrote existing IndexOffset")
	require.Equal(t, int64(3), got.MissedBlocksCounter, "seed overwrote existing MissedBlocksCounter")

	for _, consAddr := range f.consAddrs[1:] {
		info, err := f.app.SlashingKeeper.GetValidatorSigningInfo(ctx, consAddr)
		require.NoError(t, err)
		require.Equal(t, int64(0), info.StartHeight)
		require.Zero(t, info.IndexOffset)
		require.Zero(t, info.MissedBlocksCounter)
	}
}

// The original regression: a validator bonded at genesis with no signing-info
// row (the exact state simtestutil produces, and the exact state that paniced
// before the fix) must be jailable via downtime without panicking.
func TestSlashingRegression_DowntimeJailing_GenesisValidator(t *testing.T) {
	f := newSlashingFixture(t, "slashing-jail-genesis", 1, fastDowntimeParams(), func(g *slashingtypes.GenesisState, _ []sdk.ConsAddress) {
		// Explicitly blank — mimics the broken-export shape.
		g.SigningInfos = nil
		g.MissedBlocks = nil
	})

	consAddr := f.consAddrs[0]

	// Pre-check: seed populated the signing info.
	preCtx := f.ctxFactory()(1)
	_, err := f.app.SlashingKeeper.GetValidatorSigningInfo(preCtx, consAddr)
	require.NoError(t, err, "pre-condition: seed must have populated signing info")

	// Miss enough blocks to exceed maxMissed=5 past minHeight=10.
	// Loop up to 2*SignedBlocksWindow so the jail branch fires well before the end.
	simulateAbsent(t, f, consAddr, int(f.slashParams.SignedBlocksWindow)*2)

	// Post-condition: validator is jailed, signing info counters were reset.
	postCtx := f.ctxFactory()(int64(f.slashParams.SignedBlocksWindow) * 2)
	jailed, err := f.app.StakingKeeper.IsValidatorJailed(postCtx, consAddr)
	require.NoError(t, err)
	require.True(t, jailed, "validator should be jailed after sustained downtime")

	info, err := f.app.SlashingKeeper.GetValidatorSigningInfo(postCtx, consAddr)
	require.NoError(t, err)
	require.Zero(t, info.MissedBlocksCounter, "MissedBlocksCounter must reset on jail")
	require.Zero(t, info.IndexOffset, "IndexOffset must reset on jail")
}

// Multi-validator: only the offline validator is jailed; others retain
// clean signing info. Exercises the jail codepath without cross-validator
// corruption.
func TestSlashingRegression_DowntimeJailing_OnlyTargetAffected(t *testing.T) {
	f := newSlashingFixture(t, "slashing-jail-multi", 3, fastDowntimeParams(), func(g *slashingtypes.GenesisState, _ []sdk.ConsAddress) {
		g.SigningInfos = nil
		g.MissedBlocks = nil
	})

	target := f.consAddrs[0]
	bystanders := f.consAddrs[1:]

	simulateAbsent(t, f, target, int(f.slashParams.SignedBlocksWindow)*2)

	postCtx := f.ctxFactory()(int64(f.slashParams.SignedBlocksWindow) * 2)

	jailed, err := f.app.StakingKeeper.IsValidatorJailed(postCtx, target)
	require.NoError(t, err)
	require.True(t, jailed, "target must be jailed")

	for i, addr := range bystanders {
		jailed, err := f.app.StakingKeeper.IsValidatorJailed(postCtx, addr)
		require.NoErrorf(t, err, "bystander %d", i)
		require.Falsef(t, jailed, "bystander %d must not be jailed", i)

		info, err := f.app.SlashingKeeper.GetValidatorSigningInfo(postCtx, addr)
		require.NoErrorf(t, err, "bystander %d missing signing info", i)
		require.Zerof(t, info.MissedBlocksCounter, "bystander %d counter polluted", i)
	}
}

// Non-exported imports are intentionally repairable: if a forked or hand-edited
// genesis loses slashing signing infos, InitChainer should reseed them and the
// imported chain must remain slashable through the real downtime path.
func TestSlashingRegression_ExportImportForkScenario(t *testing.T) {
	src := newSlashingFixture(t, "slashing-export-src", 2, fastDowntimeParams(), nil)

	// ExportGenesis hits keepers whose state is only fully materialized after a
	// commit, so mirror the existing export tests before exporting the source app.
	commitForExport(src.app)

	exported, err := src.app.ExportAppStateAndValidators(false, nil, nil)
	require.NoError(t, err, "source export must succeed")

	var genState GenesisState
	require.NoError(t, json.Unmarshal(exported.AppState, &genState))

	// Re-import as a fresh, non-exported chain. Exported genesis stays strict and
	// is covered separately by the cross-genesis validation tests.
	var stakingGen stakingtypes.GenesisState
	src.app.AppCodec().MustUnmarshalJSON(genState[stakingtypes.ModuleName], &stakingGen)
	stakingGen.Exported = false
	genState[stakingtypes.ModuleName] = src.app.AppCodec().MustMarshalJSON(&stakingGen)

	// Simulate the artifact corruption this fix is meant to tolerate on a forked
	// import: validators remain in staking state, but slashing rows are missing.
	var slashingGen slashingtypes.GenesisState
	src.app.AppCodec().MustUnmarshalJSON(genState[slashingtypes.ModuleName], &slashingGen)
	slashingGen.SigningInfos = nil
	slashingGen.MissedBlocks = nil
	genState[slashingtypes.ModuleName] = src.app.AppCodec().MustMarshalJSON(&slashingGen)

	stateBytes, err := json.MarshalIndent(genState, "", " ")
	require.NoError(t, err)

	resetValRewardsIntegrationRuntime()
	t.Cleanup(resetValRewardsIntegrationRuntime)

	const dstChainID = "slashing-export-dst"
	dst, _ := setup(true, 5, dstChainID, config.EVMChainID)
	_, err = dst.InitChain(&abci.RequestInitChain{
		ChainId:         dstChainID,
		Validators:      []abci.ValidatorUpdate{},
		ConsensusParams: &exported.ConsensusParams,
		AppStateBytes:   stateBytes,
	})
	require.NoError(t, err, "fork import must succeed by reseeding missing slashing rows")

	dstFixture := slashingFixture{
		app:         dst,
		chainID:     dstChainID,
		validators:  src.validators,
		consAddrs:   src.consAddrs,
		slashParams: src.slashParams,
	}

	checkCtx := dstFixture.ctxFactory()(1)
	for i, consAddr := range dstFixture.consAddrs {
		info, err := dst.SlashingKeeper.GetValidatorSigningInfo(checkCtx, consAddr)
		require.NoErrorf(t, err, "validator %d missing signing info on imported chain", i)
		require.Zero(t, info.IndexOffset)
		require.Zero(t, info.MissedBlocksCounter)
	}

	target := dstFixture.consAddrs[0]
	simulateAbsent(t, dstFixture, target, int(dstFixture.slashParams.SignedBlocksWindow)*2)

	postCtx := dstFixture.ctxFactory()(int64(dstFixture.slashParams.SignedBlocksWindow) * 2)
	jailed, err := dst.StakingKeeper.IsValidatorJailed(postCtx, target)
	require.NoError(t, err)
	require.True(t, jailed, "imported validator must still be slashable via downtime")
}
