//go:build test
// +build test

package valrewards

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	evmdcmd "github.com/cosmos/evm/evmd/cmd/evmd/cmd"
	cli "github.com/cosmos/evm/evmd/tests/integration/cli"
	network "github.com/cosmos/evm/evmd/tests/network"
	vrtypes "github.com/cosmos/evm/x/valrewards/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

type governanceTestEnv struct {
	netw        *network.Network
	rootCmd     *cobra.Command
	queryClient vrtypes.QueryClient
	clientCtx1  client.Context
	clientCtx2  client.Context
	chainID     string
	feeAmount   string
	val1Addr    string
	addr2       sdk.AccAddress
}

func newGovernanceTestEnv(t *testing.T) governanceTestEnv {
	t.Helper()

	cfg := network.DefaultConfig()
	cfg.BondDenom = evmtypes.DefaultEVMDenom
	cfg.InjectDenomMetadata = true
	cfg.MinGasPrices = "0.000006" + evmtypes.DefaultEVMDenom
	cfg.NumValidators = 1
	cfg.TimeoutCommit = 200 * time.Millisecond
	cfg.APIAddress = "tcp://0.0.0.0:" + cli.FreePort(t)
	cfg.RPCAddress = "tcp://0.0.0.0:" + cli.FreePort(t)
	cfg.GRPCAddress = "0.0.0.0:" + cli.FreePort(t)
	cfg.JSONRPCAddress = "0.0.0.0:" + cli.FreePort(t)

	var govGen govv1.GenesisState
	cfg.Codec.MustUnmarshalJSON(cfg.GenesisState[govtypes.ModuleName], &govGen)
	govGen.Params.MinDeposit = sdk.NewCoins(sdk.NewCoin(cfg.BondDenom, sdkmath.NewInt(1)))
	govGen.Params.ExpeditedMinDeposit = sdk.NewCoins(sdk.NewCoin(cfg.BondDenom, sdkmath.NewInt(1)))
	votingPeriod := 5 * time.Second
	maxDepositPeriod := 5 * time.Second
	govGen.Params.VotingPeriod = &votingPeriod
	govGen.Params.MaxDepositPeriod = &maxDepositPeriod
	cfg.GenesisState[govtypes.ModuleName] = cfg.Codec.MustMarshalJSON(&govGen)

	netw, err := network.New(t, t.TempDir(), cfg)
	require.NoError(t, err)
	t.Cleanup(netw.Cleanup)

	val1 := netw.Validators[0]

	kr := val1.ClientCtx.Keyring
	keyringAlgos, _ := kr.SupportedAlgorithms()
	algo, err := keyring.NewSigningAlgoFromString(cfg.SigningAlgo, keyringAlgos)
	require.NoError(t, err)
	info, _, err := kr.NewMnemonic("user2", keyring.English, sdk.FullFundraiserPath, "", algo)
	require.NoError(t, err)
	addr2, err := info.GetAddress()
	require.NoError(t, err)

	grpcAddr := val1.AppConfig.GRPC.Address
	grpcConn, err := grpc.Dial(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = grpcConn.Close() })

	clientCtx1 := val1.ClientCtx.
		WithChainID(cfg.ChainID).
		WithNodeURI(val1.RPCAddress).
		WithHomeDir(val1.Dir).
		WithClient(val1.RPCClient).
		WithGRPCClient(grpcConn).
		WithFromName(val1.Moniker).
		WithFromAddress(val1.Address).
		WithKeyringDir(val1.ClientCtx.KeyringDir).
		WithKeyring(val1.ClientCtx.Keyring)

	clientCtx2 := val1.ClientCtx.
		WithChainID(cfg.ChainID).
		WithNodeURI(val1.RPCAddress).
		WithHomeDir(val1.Dir).
		WithClient(val1.RPCClient).
		WithGRPCClient(grpcConn).
		WithFromName("user2").
		WithFromAddress(addr2).
		WithKeyringDir(val1.ClientCtx.KeyringDir).
		WithKeyring(val1.ClientCtx.Keyring)

	_, err = netw.WaitForHeight(1)
	require.NoError(t, err)

	return governanceTestEnv{
		netw:        netw,
		rootCmd:     evmdcmd.NewRootCmd(),
		queryClient: vrtypes.NewQueryClient(clientCtx1),
		clientCtx1:  clientCtx1,
		clientCtx2:  clientCtx2,
		chainID:     cfg.ChainID,
		feeAmount:   "40000000000000000" + cfg.BondDenom,
		val1Addr:    val1.Address.String(),
		addr2:       addr2,
	}
}

func (e governanceTestEnv) fundUser2(t *testing.T) {
	t.Helper()

	fundAmount := "2000000000000000000" + evmtypes.DefaultEVMDenom
	fundOut := cli.ExecCLICmd(t, e.clientCtx1, e.rootCmd,
		"tx", "bank", "send", e.val1Addr, e.addr2.String(), fundAmount,
		"--from", e.clientCtx1.FromName,
		"--chain-id", e.chainID,
		"--fees", e.feeAmount,
		"--broadcast-mode", "sync",
		"--gas", "200000",
		"--yes",
	)
	require.Zero(t, cli.WaitForTxResultCode(t, e.netw, e.clientCtx1, cli.ExtractTxHash(t, fundOut)))
}

func (e governanceTestEnv) requireUser2SetBlocksFails(t *testing.T, blocks string) {
	t.Helper()

	out, err := cli.ExecCLICmdErr(t, e.clientCtx2, e.rootCmd,
		"tx", "valrewards", "set-blocks-in-epoch", blocks,
		"--from", "user2",
		"--chain-id", e.chainID,
		"--fees", e.feeAmount,
		"--broadcast-mode", "sync",
		"--gas", "200000",
		"--yes",
	)
	cli.RequireTxFailed(t, e.netw, e.clientCtx2, out, err)
}

func (e governanceTestEnv) requireUser2SetBlocksSucceeds(t *testing.T, blocks string) {
	t.Helper()

	out := cli.ExecCLICmd(t, e.clientCtx2, e.rootCmd,
		"tx", "valrewards", "set-blocks-in-epoch", blocks,
		"--from", "user2",
		"--chain-id", e.chainID,
		"--fees", e.feeAmount,
		"--broadcast-mode", "sync",
		"--gas", "200000",
		"--yes",
	)
	require.Zero(t, cli.WaitForTxResultCode(t, e.netw, e.clientCtx2, cli.ExtractTxHash(t, out)))
}

func (e governanceTestEnv) queryParams(t *testing.T) *vrtypes.QueryParamsResponse {
	t.Helper()

	paramsResp, err := e.queryClient.Params(context.Background(), &vrtypes.QueryParamsRequest{})
	require.NoError(t, err)
	return paramsResp
}

func (e governanceTestEnv) submitMalformedWhitelistProposal(t *testing.T, whitelist []string) {
	t.Helper()

	msg := map[string]any{
		"@type":     "/cosmos.evm.valrewards.v1.MsgUpdateParams",
		"authority": authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		"params": map[string]any{
			"whitelist": whitelist,
		},
	}
	msgBz, err := json.Marshal(msg)
	require.NoError(t, err)

	proposal := cli.GovProposalFile{
		Messages: []json.RawMessage{msgBz},
		Metadata: "",
		Title:    "Malformed valrewards whitelist proposal",
		Summary:  "invalid whitelist address",
		Deposit:  "1" + evmtypes.DefaultEVMDenom,
	}

	proposalPath := filepath.Join(t.TempDir(), "proposal.json")
	proposalBz, err := json.Marshal(proposal)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(proposalPath, proposalBz, 0o644))

	out, err := cli.ExecCLICmdErr(t, e.clientCtx1, e.rootCmd,
		"tx", "gov", "submit-proposal", proposalPath,
		"--from", e.clientCtx1.FromName,
		"--chain-id", e.chainID,
		"--fees", e.feeAmount,
		"--broadcast-mode", "sync",
		"--output", "json",
		"--gas", "200000",
		"--yes",
	)
	cli.RequireTxFailed(t, e.netw, e.clientCtx1, out, err)
}

func TestValRewardsWhitelistGovernance(t *testing.T) {
	env := newGovernanceTestEnv(t)
	env.fundUser2(t)
	env.requireUser2SetBlocksFails(t, "25")

	cli.SubmitWhitelistProposal(
		t,
		env.netw,
		env.rootCmd,
		env.clientCtx1,
		env.feeAmount,
		env.chainID,
		"/cosmos.evm.valrewards.v1.MsgUpdateParams",
		"Update valrewards whitelist",
		[]string{env.val1Addr, env.addr2.String()},
	)

	paramsResp := env.queryParams(t)
	require.Contains(t, paramsResp.Whitelist, env.addr2.String())
	require.Equal(t, vrtypes.BLOCKS_IN_EPOCH, paramsResp.CurrentRewardSettings.BlocksInEpoch)

	env.requireUser2SetBlocksSucceeds(t, "25")

	paramsResp = env.queryParams(t)
	require.True(
		t,
		paramsResp.CurrentRewardSettings.BlocksInEpoch == vrtypes.BLOCKS_IN_EPOCH || paramsResp.CurrentRewardSettings.BlocksInEpoch == 25,
	)
	require.EqualValues(t, 25, paramsResp.NextRewardSettings.BlocksInEpoch)
	require.Contains(t, paramsResp.Whitelist, env.addr2.String())
}

func TestValRewardsWhitelistGovernanceRejectedProposal(t *testing.T) {
	env := newGovernanceTestEnv(t)
	env.fundUser2(t)
	env.requireUser2SetBlocksFails(t, "25")

	cli.SubmitWhitelistProposalWithVote(
		t,
		env.netw,
		env.rootCmd,
		env.clientCtx1,
		env.feeAmount,
		env.chainID,
		"/cosmos.evm.valrewards.v1.MsgUpdateParams",
		"Reject valrewards whitelist update",
		[]string{env.val1Addr, env.addr2.String()},
		"no",
		govv1.StatusRejected,
	)

	paramsResp := env.queryParams(t)
	require.NotContains(t, paramsResp.Whitelist, env.addr2.String())
	env.requireUser2SetBlocksFails(t, "25")
}

func TestValRewardsWhitelistGovernanceReplacesWhitelist(t *testing.T) {
	env := newGovernanceTestEnv(t)
	env.fundUser2(t)

	cli.SubmitWhitelistProposal(
		t,
		env.netw,
		env.rootCmd,
		env.clientCtx1,
		env.feeAmount,
		env.chainID,
		"/cosmos.evm.valrewards.v1.MsgUpdateParams",
		"Add user2 to valrewards whitelist",
		[]string{env.val1Addr, env.addr2.String()},
	)
	env.requireUser2SetBlocksSucceeds(t, "25")

	cli.SubmitWhitelistProposal(
		t,
		env.netw,
		env.rootCmd,
		env.clientCtx1,
		env.feeAmount,
		env.chainID,
		"/cosmos.evm.valrewards.v1.MsgUpdateParams",
		"Remove user2 from valrewards whitelist",
		[]string{env.val1Addr},
	)

	paramsResp := env.queryParams(t)
	require.Contains(t, paramsResp.Whitelist, env.val1Addr)
	require.NotContains(t, paramsResp.Whitelist, env.addr2.String())
	env.requireUser2SetBlocksFails(t, "30")
}

func TestValRewardsWhitelistGovernanceRejectsMalformedProposal(t *testing.T) {
	env := newGovernanceTestEnv(t)
	env.submitMalformedWhitelistProposal(t, []string{"bad"})

	paramsResp := env.queryParams(t)
	require.Empty(t, paramsResp.Whitelist)
}
