//go:build test
// +build test

package valrewards

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

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

	net, err := network.New(t, t.TempDir(), cfg)
	require.NoError(t, err)
	t.Cleanup(net.Cleanup)

	val := net.Validators[0]

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

	rootCmd := evmdcmd.NewRootCmd()

	targetHeight := vrtypes.BLOCKS_IN_EPOCH + 2
	_, err = net.WaitForHeight(targetHeight)
	require.NoError(t, err)

	valAccAddr := val.Address.String()
	valOperAddr := val.ValAddress.String()
	epoch := "0"

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
	_ = cli.ExecCLICmd(t, clientCtx, rootCmd,
		"tx", "valrewards", "deposit", valAccAddr, depositAmount,
		"--from", val.Moniker,
		"--chain-id", cfg.ChainID,
		"--fees", feeAmount,
		"--broadcast-mode", "sync",
		"--gas", "200000",
		"--yes",
	)

	_, err = net.WaitForHeight(targetHeight + 1)
	require.NoError(t, err)

	_ = cli.ExecCLICmd(t, clientCtx, rootCmd,
		"tx", "valrewards", "claim", valAccAddr, "1", epoch,
		"--from", val.Moniker,
		"--chain-id", cfg.ChainID,
		"--fees", feeAmount,
		"--broadcast-mode", "sync",
		"--gas", "200000",
		"--yes",
	)

	_ = cli.ExecCLICmd(t, clientCtx, rootCmd, "query", "valrewards", "delegation-rewards", valAccAddr, epoch, "--output", "json")
}

func mustBigInt(v string) *big.Int {
	out, ok := new(big.Int).SetString(v, 10)
	if !ok {
		panic("invalid big int string")
	}
	return out
}
