//go:build test
// +build test

package valrewards

import (
	"context"
	"encoding/json"
	"math/big"
	"strconv"
	"strings"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	evmdcmd "github.com/cosmos/evm/evmd/cmd/evmd/cmd"
	cli "github.com/cosmos/evm/evmd/tests/integration/cli"
	network "github.com/cosmos/evm/evmd/tests/network"
	vrtypes "github.com/cosmos/evm/x/valrewards/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

type coinResp struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

type rewardsPoolResp struct {
	Pool coinResp `json:"pool"`
}

type delegationRewardsResp struct {
	Rewards coinResp `json:"rewards"`
}

func TestValRewardsCLIDemo(t *testing.T) {
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

	net, err := network.New(t, t.TempDir(), cfg)
	require.NoError(t, err)
	t.Cleanup(net.Cleanup)

	val := net.Validators[0]
	kr := val.ClientCtx.Keyring
	keyringAlgos, _ := kr.SupportedAlgorithms()
	algo, err := keyring.NewSigningAlgoFromString(cfg.SigningAlgo, keyringAlgos)
	require.NoError(t, err)
	info, _, err := kr.NewMnemonic("user2", keyring.English, sdk.FullFundraiserPath, "", algo)
	require.NoError(t, err)
	user2Addr, err := info.GetAddress()
	require.NoError(t, err)

	grpcAddr := val.AppConfig.GRPC.Address
	grpcConn, err := grpc.Dial(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = grpcConn.Close() })

	clientCtx := val.ClientCtx.
		WithChainID(cfg.ChainID).
		WithNodeURI(val.RPCAddress).
		WithHomeDir(val.Dir).
		WithClient(val.RPCClient).
		WithGRPCClient(grpcConn).
		WithFromName(val.Moniker).
		WithFromAddress(val.Address).
		WithKeyringDir(val.ClientCtx.KeyringDir).
		WithKeyring(val.ClientCtx.Keyring)
	clientCtx2 := val.ClientCtx.
		WithChainID(cfg.ChainID).
		WithNodeURI(val.RPCAddress).
		WithHomeDir(val.Dir).
		WithClient(val.RPCClient).
		WithGRPCClient(grpcConn).
		WithFromName("user2").
		WithFromAddress(user2Addr).
		WithKeyringDir(val.ClientCtx.KeyringDir).
		WithKeyring(val.ClientCtx.Keyring)

	rootCmd := evmdcmd.NewRootCmd()
	queryClient := vrtypes.NewQueryClient(clientCtx)
	bankQueryClient := banktypes.NewQueryClient(clientCtx)

	targetHeight := vrtypes.BLOCKS_IN_EPOCH + 2
	_, err = net.WaitForHeight(targetHeight)
	require.NoError(t, err)

	valAccAddr := val.Address.String()
	valOperAddr := val.ValAddress.String()
	epoch := "0"
	fundAmount := "2000000000000000000" + cfg.BondDenom

	out := cli.ExecCLICmd(t, clientCtx, rootCmd, "query", "valrewards", "rewards-pool", "--output", "json")
	var poolResp rewardsPoolResp
	require.NoError(t, json.Unmarshal([]byte(out), &poolResp))

	out = cli.ExecCLICmd(t, clientCtx, rootCmd, "query", "valrewards", "validator-outstanding-rewards", epoch, valOperAddr, "--output", "json")
	var valRewardsResp delegationRewardsResp
	require.NoError(t, json.Unmarshal([]byte(out), &valRewardsResp))
	require.NotZero(t, mustBigInt(valRewardsResp.Rewards.Amount).Sign(), "expected non-zero validator rewards")

	out = cli.ExecCLICmd(t, clientCtx, rootCmd, "query", "valrewards", "delegation-rewards", valAccAddr, epoch, "--output", "json")
	var delRewardsResp delegationRewardsResp
	require.NoError(t, json.Unmarshal([]byte(out), &delRewardsResp))
	require.NotZero(t, mustBigInt(delRewardsResp.Rewards.Amount).Sign(), "expected non-zero delegation rewards")

	feeAmount := "40000000000000000" + cfg.BondDenom
	depositAmount := "2000000000000000000" + cfg.BondDenom
	fundOut := cli.ExecCLICmd(t, clientCtx, rootCmd,
		"tx", "bank", "send", valAccAddr, user2Addr.String(), fundAmount,
		"--from", val.Moniker,
		"--chain-id", cfg.ChainID,
		"--fees", feeAmount,
		"--broadcast-mode", "sync",
		"--gas", "200000",
		"--yes",
	)
	require.Zero(t, cli.WaitForTxResultCode(t, net, clientCtx, cli.ExtractTxHash(t, fundOut)))

	depositOut := cli.ExecCLICmd(t, clientCtx, rootCmd,
		"tx", "valrewards", "deposit", depositAmount,
		"--from", val.Moniker,
		"--chain-id", cfg.ChainID,
		"--fees", feeAmount,
		"--broadcast-mode", "sync",
		"--gas", "200000",
		"--yes",
	)
	require.Zero(t, cli.WaitForTxResultCode(t, net, clientCtx, cli.ExtractTxHash(t, depositOut)))

	_, err = net.WaitForHeight(targetHeight + 1)
	require.NoError(t, err)

	claimableResp, err := queryClient.DelegationRewards(context.Background(), &vrtypes.QueryDelegationRewardsRequest{
		Delegator: valAccAddr,
		Epoch:     0,
	})
	require.NoError(t, err)
	claimAmount := claimableResp.Rewards.Amount
	require.True(t, claimAmount.IsPositive(), "expected non-zero claimable rewards before sponsored claim")

	validatorBalanceBefore, err := bankQueryClient.Balance(context.Background(), &banktypes.QueryBalanceRequest{
		Address: valAccAddr,
		Denom:   cfg.BondDenom,
	})
	require.NoError(t, err)
	sponsorBalanceBefore, err := bankQueryClient.Balance(context.Background(), &banktypes.QueryBalanceRequest{
		Address: user2Addr.String(),
		Denom:   cfg.BondDenom,
	})
	require.NoError(t, err)
	feeCoin, err := sdk.ParseCoinNormalized(feeAmount)
	require.NoError(t, err)

	claimOut := cli.ExecCLICmd(t, clientCtx2, rootCmd,
		"tx", "valrewards", "claim", valAccAddr, epoch,
		"--from", "user2",
		"--chain-id", cfg.ChainID,
		"--fees", feeAmount,
		"--broadcast-mode", "sync",
		"--gas", "200000",
		"--yes",
	)
	require.Zero(t, extractBroadcastCode(t, claimOut), "expected sponsored claim broadcast to pass: %s", claimOut)
	require.NoError(t, net.WaitForNextBlock())

	validatorBalanceAfter, err := bankQueryClient.Balance(context.Background(), &banktypes.QueryBalanceRequest{
		Address: valAccAddr,
		Denom:   cfg.BondDenom,
	})
	require.NoError(t, err)
	require.Equal(t, validatorBalanceBefore.Balance.Amount.Add(claimAmount), validatorBalanceAfter.Balance.Amount)

	sponsorBalanceAfter, err := bankQueryClient.Balance(context.Background(), &banktypes.QueryBalanceRequest{
		Address: user2Addr.String(),
		Denom:   cfg.BondDenom,
	})
	require.NoError(t, err)
	require.Equal(t, sponsorBalanceBefore.Balance.Amount.Sub(feeCoin.Amount), sponsorBalanceAfter.Balance.Amount)

	claimedResp, err := queryClient.DelegationRewards(context.Background(), &vrtypes.QueryDelegationRewardsRequest{
		Delegator: valAccAddr,
		Epoch:     0,
	})
	require.NoError(t, err)
	require.True(t, claimedResp.Rewards.IsZero(), "expected rewards to be cleared after sponsored claim")

	cli.SubmitWhitelistProposal(
		t,
		net,
		rootCmd,
		clientCtx,
		feeAmount,
		cfg.ChainID,
		"/cosmos.evm.valrewards.v1.MsgUpdateParams",
		"Update valrewards whitelist",
		[]string{valAccAddr},
	)

	setBlocksOut := cli.ExecCLICmd(t, clientCtx, rootCmd,
		"tx", "valrewards", "set-blocks-in-epoch", "25",
		"--from", val.Moniker,
		"--chain-id", cfg.ChainID,
		"--fees", feeAmount,
		"--broadcast-mode", "sync",
		"--gas", "200000",
		"--yes",
	)
	require.Zero(t, cli.WaitForTxResultCode(t, net, clientCtx, cli.ExtractTxHash(t, setBlocksOut)))

	setRewardsOut := cli.ExecCLICmd(t, clientCtx, rootCmd,
		"tx", "valrewards", "set-rewards-per-epoch", "2000000000000000000",
		"--from", val.Moniker,
		"--chain-id", cfg.ChainID,
		"--fees", feeAmount,
		"--broadcast-mode", "sync",
		"--gas", "200000",
		"--yes",
	)
	require.Zero(t, cli.WaitForTxResultCode(t, net, clientCtx, cli.ExtractTxHash(t, setRewardsOut)))

	setPausedOut := cli.ExecCLICmd(t, clientCtx, rootCmd,
		"tx", "valrewards", "set-rewarding-paused", "true",
		"--from", val.Moniker,
		"--chain-id", cfg.ChainID,
		"--fees", feeAmount,
		"--broadcast-mode", "sync",
		"--gas", "200000",
		"--yes",
	)
	require.Zero(t, cli.WaitForTxResultCode(t, net, clientCtx, cli.ExtractTxHash(t, setPausedOut)))

	_ = cli.ExecCLICmd(t, clientCtx, rootCmd, "query", "valrewards", "params", "--output", "json")
	paramsResp, err := queryClient.Params(context.Background(), &vrtypes.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, vrtypes.BLOCKS_IN_EPOCH, paramsResp.CurrentRewardSettings.BlocksInEpoch)
	require.EqualValues(t, 25, paramsResp.NextRewardSettings.BlocksInEpoch)
	require.Equal(t, vrtypes.REWARDS_PER_EPOCH, paramsResp.CurrentRewardSettings.RewardsPerEpoch)
	require.Equal(t, "2000000000000000000", paramsResp.NextRewardSettings.RewardsPerEpoch)
	require.False(t, paramsResp.CurrentRewardSettings.RewardingPaused)
	require.True(t, paramsResp.NextRewardSettings.RewardingPaused)
	require.Equal(t, vrtypes.PROPOSER_BONUS_POINTS, paramsResp.ProposerBonusPoints)
	require.Contains(t, paramsResp.Whitelist, valAccAddr)
}

func mustBigInt(v string) *big.Int {
	out, ok := new(big.Int).SetString(v, 10)
	if !ok {
		panic("invalid big int string")
	}
	return out
}

func extractBroadcastCode(t *testing.T, out string) uint32 {
	t.Helper()

	type txResp struct {
		Code uint32 `json:"code"`
	}

	var resp txResp
	if err := json.Unmarshal([]byte(out), &resp); err == nil {
		return resp.Code
	}

	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "code:") {
			continue
		}
		codeText := strings.TrimSpace(strings.TrimPrefix(line, "code:"))
		code, err := strconv.ParseUint(codeText, 10, 32)
		require.NoError(t, err, "failed to parse tx output: %s", out)
		return uint32(code)
	}

	require.FailNow(t, "failed to parse tx output", out)
	return 0
}
