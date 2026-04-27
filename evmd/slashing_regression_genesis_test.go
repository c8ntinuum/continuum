package evmd

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	cmted25519 "github.com/cometbft/cometbft/crypto/ed25519"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/evm/config"
)

// TestSlashingRegression_RealGenesisFile_NoPanicOnSlash loads the real
// genesis.json at the repo root, boots EVMD, and slashes one of the
// genesis-hardcoded validators via sustained downtime.
//
// It is the regression lock for the March 19 incident: a genesis exported with
// staking.exported=true and empty distribution per-validator arrays caused the
// AfterValidatorCreated / BeforeDelegationCreated / AfterDelegationModified
// hooks in staking.InitGenesis to be skipped. The first subsequent slash on a
// genesis validator then panicked inside distribution.decrementReferenceCount
// because ValidatorHistoricalRewards[Current.Period-1] was missing.
//
// The fixed genesis.json sets exported=false so the hooks DO fire, populating
// ValidatorCurrentRewards.Period >= 1 and the matching historical-rewards
// row. This test proves that (a) the genesis still imports cleanly, (b) the
// hooks left each validator in a well-formed distribution state, and (c) the
// BeginBlocker slash path chains through IncrementValidatorPeriod without
// panicking.
func TestSlashingRegression_RealGenesisFile_NoPanicOnSlash(t *testing.T) {
	// The VM singleton is process-global. Reset before and after to isolate
	// from any other test in this package that calls InitChain.
	resetValRewardsIntegrationRuntime()
	t.Cleanup(resetValRewardsIntegrationRuntime)

	// --- 1. Read the real genesis.json (repo root, relative to evmd/). -------
	// We deliberately do NOT use cmttypes.GenesisDocFromFile: it applies strict
	// amino-JSON conventions (all int64 fields must be quoted) that this file
	// — a normal Cosmos-SDK `ctmd export` output — does not follow. Parsing
	// only the fields we actually need (chain_id, app_state) sidesteps that.
	raw, err := os.ReadFile("../genesis.json")
	require.NoError(t, err, "cannot read genesis.json at repo root")

	var genDoc struct {
		ChainID  string          `json:"chain_id"`
		AppState json.RawMessage `json:"app_state"`
	}
	require.NoError(t, json.Unmarshal(raw, &genDoc))
	require.NotEmpty(t, genDoc.ChainID, "genesis.json missing chain_id")

	// --- 2. Unmarshal app_state so we can surgically patch slashing params. --
	var genState GenesisState
	require.NoError(t, json.Unmarshal(genDoc.AppState, &genState))

	// --- 4. Create a fresh EVMD; discard its default genesis state. -----------
	// We want to InitChain *only* with the bytes from the file.
	app, _ := setup(true, 5, genDoc.ChainID, config.EVMChainID)

	// --- 5. Patch slashing.params to fast-downtime so the test terminates. ---
	// fastDowntimeParams(): SignedBlocksWindow=10, MinSignedPerWindow=0.5.
	// Everything else (jail duration, slash fractions) stays at the file's
	// values via the decoded struct.
	var slashingGen slashingtypes.GenesisState
	app.AppCodec().MustUnmarshalJSON(genState[slashingtypes.ModuleName], &slashingGen)
	slashingGen.Params = fastDowntimeParams()
	genState[slashingtypes.ModuleName] = app.AppCodec().MustMarshalJSON(&slashingGen)

	// --- 6. InitChain with the patched state + real consensus params. --------
	stateBytes, err := json.MarshalIndent(genState, "", " ")
	require.NoError(t, err)

	// The genesis.json contains a consensus.params block, but parsing it
	// strictly is brittle (see the comment above about amino-JSON). The defaults
	// are adequate for this test: blocks are tiny, no vote extensions are in
	// use, and slashing logic doesn't consult consensus params.
	_, err = app.InitChain(&abci.RequestInitChain{
		ChainId:         genDoc.ChainID,
		Validators:      []abci.ValidatorUpdate{},
		ConsensusParams: simtestutil.DefaultConsensusParams,
		AppStateBytes:   stateBytes,
	})
	require.NoError(t, err, "InitChain must accept the real genesis.json cleanly")

	// --- 7. Extract the 3 genesis validators from the staking genesis. -------
	var stakingGen stakingtypes.GenesisState
	app.AppCodec().MustUnmarshalJSON(genState[stakingtypes.ModuleName], &stakingGen)
	require.NoError(t, stakingGen.UnpackInterfaces(app.InterfaceRegistry()),
		"staking validators carry an Any-wrapped ConsPubKey; UnpackInterfaces is required")
	require.Len(t, stakingGen.Validators, 3,
		"expected 3 genesis-hardcoded validators; got %d", len(stakingGen.Validators))

	validators := make([]*cmttypes.Validator, 0, 3)
	consAddrs := make([]sdk.ConsAddress, 0, 3)
	valAddrs := make([]sdk.ValAddress, 0, 3)

	for _, v := range stakingGen.Validators {
		sdkPk, err := v.ConsPubKey()
		require.NoError(t, err)
		// ed25519 keys use the same 32-byte encoding in both SDK and CometBFT.
		cmtPk := cmted25519.PubKey(sdkPk.Bytes())
		validators = append(validators,
			cmttypes.NewValidator(cmtPk, v.ConsensusPower(sdk.DefaultPowerReduction)))

		consBz, err := v.GetConsAddr()
		require.NoError(t, err)
		consAddrs = append(consAddrs, sdk.ConsAddress(consBz))

		valBz, err := app.StakingKeeper.ValidatorAddressCodec().StringToBytes(v.GetOperator())
		require.NoError(t, err)
		valAddrs = append(valAddrs, sdk.ValAddress(valBz))
	}

	// --- 8. Pre-flight: distribution state is healthy for every validator. --
	// This is the assertion that would have FAILED against the broken March 19
	// genesis. With exported=false the hooks fire during staking.InitGenesis
	// and each validator ends up with Current.Period=2 after its self-
	// delegation's BeforeDelegationCreated cycle. Historical[1] should have
	// ReferenceCount=2 (self-ref from Current + ref from self-delegator's
	// starting info).
	preCtx := app.NewContextLegacy(false, tmproto.Header{
		ChainID: genDoc.ChainID,
		Height:  1,
		Time:    time.Unix(0, 0).UTC(),
	})

	for i, valAddr := range valAddrs {
		cur, err := app.DistrKeeper.GetValidatorCurrentRewards(preCtx, valAddr)
		require.NoErrorf(t, err, "validator %d: GetValidatorCurrentRewards", i)
		require.GreaterOrEqualf(t, cur.Period, uint64(1),
			"validator %d: Current.Period=%d — AfterValidatorCreated/BeforeDelegationCreated "+
				"did NOT fire at InitChain (regression of the March 19 bug)",
			i, cur.Period)

		// Historical[Current.Period-1] must exist (non-zero refcount). Missing
		// records read back as zero-value with ReferenceCount==0 — exactly the
		// condition decrementReferenceCount panics on.
		hist, err := app.DistrKeeper.GetValidatorHistoricalRewards(preCtx, valAddr, cur.Period-1)
		require.NoErrorf(t, err, "validator %d: GetValidatorHistoricalRewards[%d]", i, cur.Period-1)
		require.Greaterf(t, hist.ReferenceCount, uint32(0),
			"validator %d: Historical[%d] is absent from the store (ReferenceCount=0) — "+
				"the slash-panic landmine is still present",
			i, cur.Period-1)
	}

	// --- 9. Drive the slash. -------------------------------------------------
	f := slashingFixture{
		app:         app,
		chainID:     genDoc.ChainID,
		validators:  validators,
		consAddrs:   consAddrs,
		slashParams: slashingGen.Params,
	}

	target := consAddrs[0]
	targetValAddr := valAddrs[0]

	// Snapshot Current.Period BEFORE the slash so we can assert it advanced.
	preRewards, err := app.DistrKeeper.GetValidatorCurrentRewards(preCtx, targetValAddr)
	require.NoError(t, err)
	preSlashPeriod := preRewards.Period

	// The whole point: this must not panic. simulateAbsent calls
	// slashing.BeginBlocker with crafted VoteInfos that mark `target` absent
	// for `2 * SignedBlocksWindow` blocks. At some point past minHeight, the
	// MissedBlocksCounter exceeds maxMissed and the slash + jail fires. On a
	// broken genesis this path panics inside decrementReferenceCount; on a
	// healthy genesis it completes.
	simulateAbsent(t, f, target, int(f.slashParams.SignedBlocksWindow)*2)

	// --- 10. Post-flight: jail landed, hook chain completed. -----------------
	postCtx := app.NewContextLegacy(false, tmproto.Header{
		ChainID: genDoc.ChainID,
		Height:  int64(f.slashParams.SignedBlocksWindow) * 2,
		Time:    time.Unix(0, 0).UTC(),
	})

	// 10a. Target validator is jailed.
	jailed, err := app.StakingKeeper.IsValidatorJailed(postCtx, target)
	require.NoError(t, err)
	require.True(t, jailed, "target validator must be jailed after sustained downtime")

	// 10b. Bystanders are untouched.
	for i, addr := range consAddrs[1:] {
		j, err := app.StakingKeeper.IsValidatorJailed(postCtx, addr)
		require.NoErrorf(t, err, "bystander %d", i)
		require.Falsef(t, j, "bystander %d must not be jailed", i)
	}

	// 10c. The distribution hook chain fired end-to-end. Slashing calls
	// BeforeValidatorSlashed -> updateValidatorSlashFraction ->
	// IncrementValidatorPeriod, which closes the current period and opens
	// a new one. A panic inside decrementReferenceCount would have prevented
	// the period increment; if we see it advanced, the panic did not happen.
	postRewards, err := app.DistrKeeper.GetValidatorCurrentRewards(postCtx, targetValAddr)
	require.NoError(t, err)
	require.Greater(t, postRewards.Period, preSlashPeriod,
		"target validator's Current.Period must have advanced past %d — "+
			"slash hook chain must complete without panicking", preSlashPeriod)
}

// TestSlashingRegression_RealGenesisFile_DoubleSignNoPanic is the symmetric
// regression for double-signing (equivocation). It loads the real genesis.json,
// boots EVMD, and pushes a DUPLICATE_VOTE misbehavior through a FinalizeBlock
// call. The evidence module's BeginBlocker picks up the misbehavior, calls
// handleEquivocationEvidence -> StakingKeeper.Slash(INFRACTION_DOUBLE_SIGN)
// -> BeforeValidatorSlashed hook -> distribution.updateValidatorSlashFraction
// -> IncrementValidatorPeriod -> decrementReferenceCount. That's the same hook
// chain that killed the chain on April 18 under downtime slashing; this test
// proves it is equally safe under the double-sign trigger.
//
// Unlike the downtime test, this one exercises the full ABCI path
// (FinalizeBlock + Commit) rather than calling slashing.BeginBlocker
// directly — `handleEquivocationEvidence` is unexported in cosmos-sdk
// v0.53.6, so a direct keeper call isn't available.
func TestSlashingRegression_RealGenesisFile_DoubleSignNoPanic(t *testing.T) {
	resetValRewardsIntegrationRuntime()
	t.Cleanup(resetValRewardsIntegrationRuntime)

	// --- 1. Load genesis.json, InitChain. (same setup as the downtime test) --
	raw, err := os.ReadFile("../genesis.json")
	require.NoError(t, err, "cannot read genesis.json at repo root")

	var genDoc struct {
		ChainID  string          `json:"chain_id"`
		AppState json.RawMessage `json:"app_state"`
	}
	require.NoError(t, json.Unmarshal(raw, &genDoc))
	require.NotEmpty(t, genDoc.ChainID, "genesis.json missing chain_id")

	var genState GenesisState
	require.NoError(t, json.Unmarshal(genDoc.AppState, &genState))

	app, _ := setup(true, 5, genDoc.ChainID, config.EVMChainID)

	// For double-sign we intentionally do NOT override slashing params: the
	// slash fires from a single misbehavior event (not missed blocks), so the
	// file's real SlashFractionDoubleSign / DowntimeJailDuration values apply.
	stateBytes, err := json.MarshalIndent(genState, "", " ")
	require.NoError(t, err)

	_, err = app.InitChain(&abci.RequestInitChain{
		ChainId:         genDoc.ChainID,
		Validators:      []abci.ValidatorUpdate{},
		ConsensusParams: simtestutil.DefaultConsensusParams,
		AppStateBytes:   stateBytes,
	})
	require.NoError(t, err, "InitChain must accept the real genesis.json cleanly")

	// --- 2. Extract validators (same shape as downtime test). ---------------
	var stakingGen stakingtypes.GenesisState
	app.AppCodec().MustUnmarshalJSON(genState[stakingtypes.ModuleName], &stakingGen)
	require.NoError(t, stakingGen.UnpackInterfaces(app.InterfaceRegistry()))
	require.Len(t, stakingGen.Validators, 3)

	validators := make([]*cmttypes.Validator, 0, 3)
	consAddrs := make([]sdk.ConsAddress, 0, 3)
	valAddrs := make([]sdk.ValAddress, 0, 3)

	for _, v := range stakingGen.Validators {
		sdkPk, err := v.ConsPubKey()
		require.NoError(t, err)
		cmtPk := cmted25519.PubKey(sdkPk.Bytes())
		validators = append(validators,
			cmttypes.NewValidator(cmtPk, v.ConsensusPower(sdk.DefaultPowerReduction)))

		consBz, err := v.GetConsAddr()
		require.NoError(t, err)
		consAddrs = append(consAddrs, sdk.ConsAddress(consBz))

		valBz, err := app.StakingKeeper.ValidatorAddressCodec().StringToBytes(v.GetOperator())
		require.NoError(t, err)
		valAddrs = append(valAddrs, sdk.ValAddress(valBz))
	}

	// --- 3. Pre-flight: distribution state healthy. -------------------------
	preCtx := app.NewContextLegacy(false, tmproto.Header{
		ChainID: genDoc.ChainID,
		Height:  1,
		Time:    time.Unix(0, 0).UTC(),
	})
	for i, valAddr := range valAddrs {
		cur, err := app.DistrKeeper.GetValidatorCurrentRewards(preCtx, valAddr)
		require.NoErrorf(t, err, "validator %d: GetValidatorCurrentRewards", i)
		require.GreaterOrEqualf(t, cur.Period, uint64(1),
			"validator %d: Current.Period=%d — hooks did NOT fire at InitChain", i, cur.Period)
		hist, err := app.DistrKeeper.GetValidatorHistoricalRewards(preCtx, valAddr, cur.Period-1)
		require.NoErrorf(t, err, "validator %d: GetValidatorHistoricalRewards[%d]", i, cur.Period-1)
		require.Greaterf(t, hist.ReferenceCount, uint32(0),
			"validator %d: Historical[%d] is absent from the store", i, cur.Period-1)
	}

	// --- 4. Drive double-sign via FinalizeBlock + Misbehavior. ---------------
	// We run two FinalizeBlock calls: block 1 normal, block 2 reports the
	// double-sign that occurred at block 1. This mirrors the real chain where
	// evidence is submitted on a later block than the one it concerns.

	target := consAddrs[0]
	targetValAddr := valAddrs[0]

	preRewards, err := app.DistrKeeper.GetValidatorCurrentRewards(preCtx, targetValAddr)
	require.NoError(t, err)
	preSlashPeriod := preRewards.Period

	// DecidedLastCommit with every validator marked present. Using validator[1]
	// as proposer so the target (which we are about to slash) is not also
	// driving fee allocation during the slashing block.
	votesAll := make([]abci.VoteInfo, 0, len(validators))
	for _, v := range validators {
		votesAll = append(votesAll, abci.VoteInfo{
			Validator: abci.Validator{
				Address: v.Address.Bytes(),
				Power:   v.VotingPower,
			},
			BlockIdFlag: tmproto.BlockIDFlagCommit,
		})
	}
	var totalPower int64
	for _, v := range validators {
		totalPower += v.VotingPower
	}
	proposer := validators[1].Address.Bytes()

	blockTime1 := time.Unix(0, 0).UTC()
	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height:            1,
		Time:              blockTime1,
		DecidedLastCommit: abci.CommitInfo{Votes: votesAll},
		ProposerAddress:   proposer,
	})
	require.NoError(t, err, "FinalizeBlock(1) must succeed before evidence block")
	_, err = app.Commit()
	require.NoError(t, err)

	blockTime2 := blockTime1.Add(time.Second)
	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height:            2,
		Time:              blockTime2,
		DecidedLastCommit: abci.CommitInfo{Votes: votesAll},
		ProposerAddress:   proposer,
		Misbehavior: []abci.Misbehavior{{
			Type: abci.MisbehaviorType_DUPLICATE_VOTE,
			Validator: abci.Validator{
				Address: target.Bytes(),
				Power:   validators[0].VotingPower,
			},
			Height:           1,
			Time:             blockTime1,
			TotalVotingPower: totalPower,
		}},
	})
	require.NoError(t, err, "FinalizeBlock(2) with DUPLICATE_VOTE misbehavior must not panic")
	_, err = app.Commit()
	require.NoError(t, err)

	// --- 5. Post-flight assertions. -----------------------------------------
	// NewContextLegacy(false, ...) reads from finalizeBlockState, which is
	// nil after Commit. Use check mode (true) instead — it reads from
	// checkState, which baseapp refreshes on every Commit.
	postCtx := app.NewContextLegacy(true, tmproto.Header{
		ChainID: genDoc.ChainID,
		Height:  3,
		Time:    blockTime2.Add(time.Second),
	})

	// 5a. Target validator is jailed.
	jailed, err := app.StakingKeeper.IsValidatorJailed(postCtx, target)
	require.NoError(t, err)
	require.True(t, jailed, "target must be jailed after double-sign")

	// 5b. Target validator is tombstoned — distinguishing double-sign from
	// downtime. Tombstoning is permanent and applies only for equivocation.
	signingInfo, err := app.SlashingKeeper.GetValidatorSigningInfo(postCtx, target)
	require.NoError(t, err)
	require.True(t, signingInfo.Tombstoned, "target must be tombstoned after double-sign")

	// 5c. Bystanders are untouched.
	for i, addr := range consAddrs[1:] {
		j, err := app.StakingKeeper.IsValidatorJailed(postCtx, addr)
		require.NoErrorf(t, err, "bystander %d", i)
		require.Falsef(t, j, "bystander %d must not be jailed", i)
	}

	// 5d. Distribution hook chain completed — the very thing that panicked
	// on April 18. If Current.Period advanced, IncrementValidatorPeriod
	// ran through decrementReferenceCount without panicking.
	postRewards, err := app.DistrKeeper.GetValidatorCurrentRewards(postCtx, targetValAddr)
	require.NoError(t, err)
	require.Greater(t, postRewards.Period, preSlashPeriod,
		"target validator's Current.Period must have advanced past %d — "+
			"double-sign slash hook chain must complete without panicking", preSlashPeriod)
}
