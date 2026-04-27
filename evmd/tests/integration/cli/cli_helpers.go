//go:build test
// +build test

package cli

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	network "github.com/cosmos/evm/evmd/tests/network"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdktestutilcli "github.com/cosmos/cosmos-sdk/testutil/cli"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

type GovProposalFile struct {
	Messages []json.RawMessage `json:"messages"`
	Metadata string            `json:"metadata"`
	Title    string            `json:"title"`
	Summary  string            `json:"summary"`
	Deposit  string            `json:"deposit"`
}

func SubmitWhitelistProposal(
	t *testing.T,
	netw *network.Network,
	rootCmd *cobra.Command,
	clientCtx client.Context,
	feeAmount string,
	chainID string,
	msgType string,
	title string,
	whitelist []string,
) uint64 {
	return SubmitWhitelistProposalWithVote(
		t,
		netw,
		rootCmd,
		clientCtx,
		feeAmount,
		chainID,
		msgType,
		title,
		whitelist,
		"yes",
		govv1.StatusPassed,
	)
}

func SubmitWhitelistProposalWithVote(
	t *testing.T,
	netw *network.Network,
	rootCmd *cobra.Command,
	clientCtx client.Context,
	feeAmount string,
	chainID string,
	msgType string,
	title string,
	whitelist []string,
	voteOption string,
	expectedStatus govv1.ProposalStatus,
) uint64 {
	t.Helper()

	msg := map[string]any{
		"@type":     msgType,
		"authority": authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		"params": map[string]any{
			"whitelist": whitelist,
		},
	}
	msgBz, err := json.Marshal(msg)
	require.NoError(t, err)

	proposal := GovProposalFile{
		Messages: []json.RawMessage{msgBz},
		Metadata: "",
		Title:    title,
		Summary:  "add whitelist entries",
		Deposit:  "1" + netw.Config.BondDenom,
	}

	proposalPath := filepath.Join(t.TempDir(), "proposal.json")
	proposalBz, err := json.Marshal(proposal)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(proposalPath, proposalBz, 0o644))

	govQuery := govv1.NewQueryClient(clientCtx)
	prevMaxID, _ := queryLatestProposalIDMaybeGRPC(t, govQuery)
	submitOut := ExecCLICmd(t, clientCtx, rootCmd,
		"tx", "gov", "submit-proposal", proposalPath,
		"--from", clientCtx.FromName,
		"--chain-id", chainID,
		"--fees", feeAmount,
		"--broadcast-mode", "sync",
		"--output", "json",
		"--gas", "200000",
		"--yes",
	)
	if code, ok := extractBroadcastCode(submitOut); ok {
		require.Zerof(t, code, "submit proposal broadcast failed: %s", submitOut)
	}
	submitTxHash := ExtractTxHash(t, submitOut)
	require.Zerof(t, WaitForTxResultCode(t, netw, clientCtx, submitTxHash), "submit proposal failed: %s", submitOut)

	var proposalID uint64
	for i := 0; i < 20; i++ {
		maxID, ok := queryLatestProposalIDMaybeGRPC(t, govQuery)
		if ok && maxID > prevMaxID {
			proposalID = maxID
			break
		}
		require.NoError(t, netw.WaitForNextBlock())
	}
	require.NotZero(t, proposalID)

	depositOut := ExecCLICmd(t, clientCtx, rootCmd,
		"tx", "gov", "deposit", strconv.FormatUint(proposalID, 10), "1"+netw.Config.BondDenom,
		"--from", clientCtx.FromName,
		"--chain-id", chainID,
		"--fees", feeAmount,
		"--broadcast-mode", "sync",
		"--output", "json",
		"--gas", "200000",
		"--yes",
	)
	require.Zerof(t, WaitForTxResultCode(t, netw, clientCtx, ExtractTxHash(t, depositOut)), "deposit proposal failed: %s", depositOut)
	require.NoError(t, netw.WaitForNextBlock())

	voteOut := ExecCLICmd(t, clientCtx, rootCmd,
		"tx", "gov", "vote", strconv.FormatUint(proposalID, 10), voteOption,
		"--from", clientCtx.FromName,
		"--chain-id", chainID,
		"--fees", feeAmount,
		"--broadcast-mode", "sync",
		"--output", "json",
		"--gas", "200000",
		"--yes",
	)
	require.Zerof(t, WaitForTxResultCode(t, netw, clientCtx, ExtractTxHash(t, voteOut)), "vote proposal failed: %s", voteOut)
	require.NoError(t, netw.WaitForNextBlock())

	lastStatus := govv1.StatusNil
	lastReason := ""
	var lastTally *govv1.TallyResult
	finalized := false
	for i := 0; i < 50; i++ {
		require.NoError(t, netw.WaitForNextBlock())
		resp, err := govQuery.Proposal(context.Background(), &govv1.QueryProposalRequest{ProposalId: proposalID})
		require.NoError(t, err)
		lastStatus = resp.Proposal.Status
		lastReason = resp.Proposal.FailedReason
		lastTally = resp.Proposal.FinalTallyResult
		if isFinalGovStatus(resp.Proposal.Status) {
			finalized = true
			break
		}
	}
	require.Truef(t, finalized, "proposal did not finalize, last status: %s, reason: %s, tally: %+v", lastStatus.String(), lastReason, lastTally)
	require.Equalf(t, expectedStatus, lastStatus, "unexpected proposal status, reason: %s, tally: %+v", lastReason, lastTally)

	return proposalID
}

func isFinalGovStatus(status govv1.ProposalStatus) bool {
	switch status {
	case govv1.StatusPassed, govv1.StatusRejected, govv1.StatusFailed:
		return true
	default:
		return false
	}
}

func queryLatestProposalIDMaybeGRPC(t *testing.T, govQuery govv1.QueryClient) (uint64, bool) {
	t.Helper()

	resp, err := govQuery.Proposals(context.Background(), &govv1.QueryProposalsRequest{})
	require.NoError(t, err)
	if len(resp.Proposals) == 0 {
		return 0, false
	}
	var maxID uint64
	for _, p := range resp.Proposals {
		if p.Id > maxID {
			maxID = p.Id
		}
	}
	return maxID, true
}

func ExecCLICmd(t *testing.T, clientCtx client.Context, rootCmd *cobra.Command, args ...string) string {
	t.Helper()
	ensureHomeFlag(t, rootCmd, clientCtx.HomeDir)
	rootCmd.SetIn(strings.NewReader("y\n"))
	args = ensureNodeArg(args, clientCtx.NodeURI)
	if len(args) > 0 && args[0] == "tx" {
		args = append(args, "--keyring-dir", clientCtx.KeyringDir)
		args = append(args, "--keyring-backend", "test")
	}
	var outBuf testutil.BufferWriter
	captured, err := captureStdoutStderr(t, func() error {
		var err error
		outBuf, err = sdktestutilcli.ExecTestCLICmd(clientCtx, rootCmd, args)
		return err
	})
	require.NoError(t, err)
	return mergeCLIOutput(outBuf.String(), captured)
}

func ExecCLICmdErr(t *testing.T, clientCtx client.Context, rootCmd *cobra.Command, args ...string) (string, error) {
	t.Helper()
	ensureHomeFlag(t, rootCmd, clientCtx.HomeDir)
	rootCmd.SetIn(strings.NewReader("y\n"))
	args = ensureNodeArg(args, clientCtx.NodeURI)
	if len(args) > 0 && args[0] == "tx" {
		args = append(args, "--keyring-dir", clientCtx.KeyringDir)
		args = append(args, "--keyring-backend", "test")
	}
	var outBuf testutil.BufferWriter
	captured, err := captureStdoutStderr(t, func() error {
		var err error
		outBuf, err = sdktestutilcli.ExecTestCLICmd(clientCtx, rootCmd, args)
		return err
	})
	return mergeCLIOutput(outBuf.String(), captured), err
}

func ensureHomeFlag(t *testing.T, cmd *cobra.Command, home string) {
	t.Helper()
	if cmd.PersistentFlags().Lookup(flags.FlagHome) == nil {
		cmd.PersistentFlags().String(flags.FlagHome, home, "home directory")
	}
	require.NoError(t, cmd.PersistentFlags().Set(flags.FlagHome, home))
}

func ensureNodeArg(args []string, nodeURI string) []string {
	for i, arg := range args {
		if arg == "--"+flags.FlagNode && i+1 < len(args) {
			return args
		}
		if strings.HasPrefix(arg, "--"+flags.FlagNode+"=") {
			return args
		}
	}
	return append(args, "--"+flags.FlagNode, nodeURI)
}

func ExtractTxHash(t *testing.T, out string) string {
	t.Helper()
	type txResp struct {
		TxHash string `json:"txhash"`
	}
	var resp txResp
	if err := json.Unmarshal([]byte(out), &resp); err == nil && resp.TxHash != "" {
		return resp.TxHash
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if idx := strings.Index(line, "txhash:"); idx >= 0 {
			return strings.TrimSpace(strings.TrimPrefix(line[idx:], "txhash:"))
		}
		var lineResp txResp
		if err := json.Unmarshal([]byte(line), &lineResp); err == nil && lineResp.TxHash != "" {
			return lineResp.TxHash
		}
	}
	t.Fatalf("failed to extract txhash from output: %s", out)
	return ""
}

func RequireTxFailed(t *testing.T, netw *network.Network, clientCtx client.Context, out string, err error) {
	t.Helper()
	if err != nil {
		return
	}

	// CLI sync broadcast may already include a non-zero CheckTx code.
	if code, ok := extractBroadcastCode(out); ok && code != 0 {
		return
	}

	txHash := ExtractTxHash(t, out)
	code, found, queryErr := WaitForTxResultCodeMaybe(t, netw, clientCtx, txHash)
	if queryErr != nil {
		// Not found after polling is valid for CheckTx-rejected txs that never enter blocks.
		if strings.Contains(strings.ToLower(queryErr.Error()), "not found") {
			return
		}
		require.NoError(t, queryErr)
	}
	if !found {
		return
	}
	if code != 0 {
		return
	}
	t.Fatalf("expected tx failure but got success; output: %s", out)
}

func captureStdoutStderr(t *testing.T, fn func() error) (string, error) {
	t.Helper()

	origStdout := os.Stdout
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = w
	os.Stderr = w

	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()

	runErr := fn()
	_ = w.Close()
	os.Stdout = origStdout
	os.Stderr = origStderr
	captured := <-done
	_ = r.Close()

	return captured, runErr
}

func mergeCLIOutput(parts ...string) string {
	var b strings.Builder
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(p)
	}
	return b.String()
}

func WaitForTx(t *testing.T, netw *network.Network, clientCtx client.Context, txHash string) {
	t.Helper()
	txHash = strings.TrimPrefix(txHash, "0x")
	hashBytes, err := hex.DecodeString(txHash)
	require.NoError(t, err)
	var lastErr error
	for i := 0; i < 20; i++ {
		_, err := clientCtx.Client.Tx(context.Background(), hashBytes, false)
		if err == nil {
			return
		}
		lastErr = err
		require.NoError(t, netw.WaitForNextBlock())
	}
	require.NoError(t, lastErr)
}

func WaitForTxResultCode(t *testing.T, netw *network.Network, clientCtx client.Context, txHash string) uint32 {
	t.Helper()
	code, found, err := WaitForTxResultCodeMaybe(t, netw, clientCtx, txHash)
	require.NoError(t, err)
	require.True(t, found, "tx %s not found", txHash)
	return code
}

func WaitForTxResultCodeMaybe(t *testing.T, netw *network.Network, clientCtx client.Context, txHash string) (uint32, bool, error) {
	t.Helper()
	txHash = strings.TrimPrefix(txHash, "0x")
	hashBytes, err := hex.DecodeString(txHash)
	require.NoError(t, err)
	var lastErr error
	for i := 0; i < 20; i++ {
		res, err := clientCtx.Client.Tx(context.Background(), hashBytes, false)
		if err == nil {
			return res.TxResult.Code, true, nil
		}
		lastErr = err
		require.NoError(t, netw.WaitForNextBlock())
	}
	return 0, false, lastErr
}

func extractBroadcastCode(out string) (uint32, bool) {
	type txResp struct {
		Code uint32 `json:"code"`
	}

	var resp txResp
	if err := json.Unmarshal([]byte(out), &resp); err == nil {
		return resp.Code, true
	}

	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var lineResp txResp
		if err := json.Unmarshal([]byte(line), &lineResp); err == nil {
			return lineResp.Code, true
		}
	}

	return 0, false
}

func FreePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()

	addr := l.Addr().(*net.TCPAddr)
	return strconv.Itoa(addr.Port)
}
