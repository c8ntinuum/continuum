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
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/evm/config"
	valrewardstypes "github.com/cosmos/evm/x/valrewards/types"
)

// genesisMutator mutates the parsed GenesisState in-place before it is
// re-marshalled and fed to InitChain. Used to override parameters (e.g.
// blocks_in_epoch, signing window) for tests that need a small value.
type genesisMutator func(t *testing.T, app *EVMD, genState GenesisState)

// realGenesisFixture is the boot-level fixture produced by loadRealGenesisFixture:
// an EVMD app with the real genesis.json already applied, plus the
// validator handles needed to drive FinalizeBlock / assert post-state.
type realGenesisFixture struct {
	app        *EVMD
	chainID    string
	validators []*cmttypes.Validator
	consAddrs  []sdk.ConsAddress
	valAddrs   []sdk.ValAddress
}

// loadRealGenesisFixture loads /genesis.json (repo root), applies any caller
// mutators (param overrides, etc.), calls InitChain, and returns the booted
// app plus the validator set in a form ready for FinalizeBlock.
//
// Mutators see the decoded GenesisState and can edit any module's sub-section
// before the final marshal + InitChain. Mutators must use the app's codec
// (AppCodec().MustUnmarshalJSON / MustMarshalJSON) so custom types round-trip
// cleanly.
func loadRealGenesisFixture(t *testing.T, mutators ...genesisMutator) realGenesisFixture {
	t.Helper()

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

	for _, m := range mutators {
		m(t, app, genState)
	}

	stateBytes, err := json.MarshalIndent(genState, "", " ")
	require.NoError(t, err)

	_, err = app.InitChain(&abci.RequestInitChain{
		ChainId:         genDoc.ChainID,
		Validators:      []abci.ValidatorUpdate{},
		ConsensusParams: simtestutil.DefaultConsensusParams,
		AppStateBytes:   stateBytes,
	})
	require.NoError(t, err, "InitChain must accept the real genesis.json cleanly")

	var stakingGen stakingtypes.GenesisState
	app.AppCodec().MustUnmarshalJSON(genState[stakingtypes.ModuleName], &stakingGen)
	require.NoError(t, stakingGen.UnpackInterfaces(app.InterfaceRegistry()))
	require.Len(t, stakingGen.Validators, 3, "expected 3 genesis-hardcoded validators")

	fixture := realGenesisFixture{
		app:        app,
		chainID:    genDoc.ChainID,
		validators: make([]*cmttypes.Validator, 0, 3),
		consAddrs:  make([]sdk.ConsAddress, 0, 3),
		valAddrs:   make([]sdk.ValAddress, 0, 3),
	}
	for _, v := range stakingGen.Validators {
		sdkPk, err := v.ConsPubKey()
		require.NoError(t, err)
		cmtPk := cmted25519.PubKey(sdkPk.Bytes())
		fixture.validators = append(fixture.validators,
			cmttypes.NewValidator(cmtPk, v.ConsensusPower(sdk.DefaultPowerReduction)))

		consBz, err := v.GetConsAddr()
		require.NoError(t, err)
		fixture.consAddrs = append(fixture.consAddrs, sdk.ConsAddress(consBz))

		valBz, err := app.StakingKeeper.ValidatorAddressCodec().StringToBytes(v.GetOperator())
		require.NoError(t, err)
		fixture.valAddrs = append(fixture.valAddrs, sdk.ValAddress(valBz))
	}
	return fixture
}

// allVotes returns a DecidedLastCommit with every validator marked present.
// Used for the FinalizeBlock plumbing in the boot / epoch tests.
func (f realGenesisFixture) allVotes() []abci.VoteInfo {
	votes := make([]abci.VoteInfo, 0, len(f.validators))
	for _, v := range f.validators {
		votes = append(votes, abci.VoteInfo{
			Validator: abci.Validator{
				Address: v.Address.Bytes(),
				Power:   v.VotingPower,
			},
			BlockIdFlag: tmproto.BlockIDFlagCommit,
		})
	}
	return votes
}

// advanceBlock runs one FinalizeBlock + Commit at the given height with every
// validator present and validators[height % N] as proposer. Shared between the
// boot test and the epoch-rollover test.
func (f realGenesisFixture) advanceBlock(t *testing.T, height int64, blockTime time.Time) {
	t.Helper()
	proposer := f.validators[int(height-1)%len(f.validators)].Address.Bytes()
	_, err := f.app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height:            height,
		Time:              blockTime,
		DecidedLastCommit: abci.CommitInfo{Votes: f.allVotes()},
		ProposerAddress:   proposer,
	})
	require.NoErrorf(t, err, "FinalizeBlock(%d) must not panic", height)
	_, err = f.app.Commit()
	require.NoErrorf(t, err, "Commit(%d) must succeed", height)
}

// TestRealGenesisFile_BootAndAdvance5Blocks loads the real genesis.json,
// InitChains the app, and runs 5 FinalizeBlock + Commit cycles with rotating
// proposers. This is the broadest-possible single-test regression net: any
// module whose BeginBlocker or EndBlocker panics on this specific genesis
// shape fails here. In particular it exercises modules that don't have
// dedicated regressions — x/erc20, x/precisebank, x/feemarket, x/circuit,
// x/ibcbreaker, x/ibcratelimiterext — all of which run on every block.
func TestRealGenesisFile_BootAndAdvance5Blocks(t *testing.T) {
	resetValRewardsIntegrationRuntime()
	t.Cleanup(resetValRewardsIntegrationRuntime)

	f := loadRealGenesisFixture(t)

	blockTime := time.Unix(0, 0).UTC()
	const blocks = 5
	for h := int64(1); h <= blocks; h++ {
		f.advanceBlock(t, h, blockTime.Add(time.Duration(h)*time.Second))
	}

	// Post-conditions: all validators still bonded, none jailed, none
	// tombstoned. Nothing should have happened to the validator set during
	// a clean 5-block run.
	postCtx := f.app.NewContextLegacy(true, tmproto.Header{
		ChainID: f.chainID,
		Height:  blocks + 1,
		Time:    blockTime.Add(time.Duration(blocks+1) * time.Second),
	})
	for i, consAddr := range f.consAddrs {
		validator, err := f.app.StakingKeeper.GetValidatorByConsAddr(postCtx, consAddr)
		require.NoErrorf(t, err, "validator %d: GetValidatorByConsAddr", i)
		require.Equalf(t, stakingtypes.Bonded, validator.Status,
			"validator %d unexpectedly left bonded set after %d clean blocks", i, blocks)

		jailed, err := f.app.StakingKeeper.IsValidatorJailed(postCtx, consAddr)
		require.NoErrorf(t, err, "validator %d: IsValidatorJailed", i)
		require.Falsef(t, jailed, "validator %d unexpectedly jailed after %d clean blocks", i, blocks)

		info, err := f.app.SlashingKeeper.GetValidatorSigningInfo(postCtx, consAddr)
		require.NoErrorf(t, err, "validator %d: GetValidatorSigningInfo", i)
		require.Falsef(t, info.Tombstoned, "validator %d unexpectedly tombstoned", i)
	}

	// Height really advanced to `blocks` (not, say, stuck on 1 because
	// nothing was committing).
	require.Equal(t, int64(blocks), f.app.LastBlockHeight(),
		"chain height must be %d after %d FinalizeBlock+Commit cycles", blocks, blocks)
}

// TestRealGenesisFile_ValRewardsEpochRollover specifically exercises
// x/valrewards.BeginBlocker's epoch-transition path, which runs on the block
// where BlocksIntoCurrentEpoch reaches CurrentRewardSettings.BlocksInEpoch.
// A divide-by-zero in params, a missing epoch_state, or a malformed
// ValidatorPoints record would halt the chain on exactly that block — this
// test is the only place the transition is exercised end-to-end against the
// real genesis.
//
// The real genesis has blocks_in_epoch=21600, far too many for a test. We
// override it to a small value via a genesisMutator so the rollover fires
// after ~6 blocks.
func TestRealGenesisFile_ValRewardsEpochRollover(t *testing.T) {
	resetValRewardsIntegrationRuntime()
	t.Cleanup(resetValRewardsIntegrationRuntime)

	// valrewards params validation enforces MinBlocksInEpoch=20
	// (see x/valrewards/types/params.go:13). Use the minimum.
	const testBlocksInEpoch = 20

	shrinkEpoch := func(t *testing.T, _ *EVMD, genState GenesisState) {
		t.Helper()
		// valrewards uses plain encoding/json (see x/valrewards/module.go:143),
		// not the app codec, so we do the same here.
		var vrGen valrewardstypes.GenesisState
		require.NoError(t, json.Unmarshal(genState[valrewardstypes.ModuleName], &vrGen))
		vrGen.CurrentRewardSettings.BlocksInEpoch = testBlocksInEpoch
		vrGen.NextRewardSettings.BlocksInEpoch = testBlocksInEpoch
		patched, err := json.Marshal(&vrGen)
		require.NoError(t, err)
		genState[valrewardstypes.ModuleName] = patched
	}

	f := loadRealGenesisFixture(t, shrinkEpoch)

	// Read the starting epoch state so we can assert it advanced, not just
	// that we avoided a panic.
	preCtx := f.app.NewContextLegacy(false, tmproto.Header{
		ChainID: f.chainID,
		Height:  1,
		Time:    time.Unix(0, 0).UTC(),
	})
	preEpoch := f.app.ValRewardsKeeper.GetEpochState(preCtx)

	// Advance 2 * BlocksInEpoch blocks to cross the boundary twice. One
	// rollover is sufficient to hit the bug class, but two catches any
	// off-by-one in the counter reset.
	blockTime := time.Unix(0, 0).UTC()
	totalBlocks := int64(2 * testBlocksInEpoch)
	for h := int64(1); h <= totalBlocks; h++ {
		f.advanceBlock(t, h, blockTime.Add(time.Duration(h)*time.Second))
	}

	// Post-conditions.
	postCtx := f.app.NewContextLegacy(true, tmproto.Header{
		ChainID: f.chainID,
		Height:  totalBlocks + 1,
		Time:    blockTime.Add(time.Duration(totalBlocks+1) * time.Second),
	})
	postEpoch := f.app.ValRewardsKeeper.GetEpochState(postCtx)

	// 1. The chain committed every block (no silent BeginBlocker panic
	//    swallowed by retry logic).
	require.Equal(t, totalBlocks, f.app.LastBlockHeight(),
		"chain must advance through %d blocks across epoch boundary", totalBlocks)

	// 2. Epoch counter moved forward — the rollover actually fired. For
	//    2*BlocksInEpoch blocks we expect CurrentEpoch to advance by at
	//    least 1 from wherever it started (the legacy fallback may have
	//    put it at a non-zero initial value based on BlockHeight).
	require.Greater(t, postEpoch.CurrentEpoch, preEpoch.CurrentEpoch,
		"valrewards epoch must advance after crossing BlocksInEpoch boundary "+
			"(pre=%d post=%d)", preEpoch.CurrentEpoch, postEpoch.CurrentEpoch)

	// 3. Validators still healthy — epoch-rollover side-effects must not
	//    slash or jail anyone.
	for i, consAddr := range f.consAddrs {
		jailed, err := f.app.StakingKeeper.IsValidatorJailed(postCtx, consAddr)
		require.NoError(t, err)
		require.Falsef(t, jailed, "validator %d must not be jailed by epoch rollover", i)
	}
}
