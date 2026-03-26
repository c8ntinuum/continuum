//go:build test

package testutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govmodule "github.com/cosmos/cosmos-sdk/x/gov"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

func DefaultGovGenesis(baseDenom string) *govv1.GenesisState {
	govGen := govv1.DefaultGenesisState()
	govGen.Params.MinDeposit = sdk.NewCoins(sdk.NewCoin(baseDenom, sdkmath.NewInt(1)))
	govGen.Params.ExpeditedMinDeposit = sdk.NewCoins(sdk.NewCoin(baseDenom, sdkmath.NewInt(1)))
	votingPeriod := time.Second
	maxDepositPeriod := time.Second
	govGen.Params.VotingPeriod = &votingPeriod
	govGen.Params.MaxDepositPeriod = &maxDepositPeriod
	return govGen
}

func SubmitAndPassProposal(
	t *testing.T,
	ctx sdk.Context,
	govKeeper govkeeper.Keeper,
	baseDenom string,
	proposer sdk.AccAddress,
	msg sdk.Msg,
	title string,
	summary string,
) sdk.Context {
	t.Helper()

	proposal, err := govKeeper.SubmitProposal(ctx, []sdk.Msg{msg}, "", title, summary, proposer, false)
	require.NoError(t, err)

	votingStarted, err := govKeeper.AddDeposit(ctx, proposal.Id, proposer, sdk.NewCoins(sdk.NewCoin(baseDenom, sdkmath.NewInt(1))))
	require.NoError(t, err)
	require.True(t, votingStarted)

	err = govKeeper.AddVote(ctx, proposal.Id, proposer, govv1.NewNonSplitVoteOption(govv1.OptionYes), "")
	require.NoError(t, err)

	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(2 * time.Second))
	require.NoError(t, govmodule.EndBlocker(ctx, &govKeeper))

	return ctx
}
