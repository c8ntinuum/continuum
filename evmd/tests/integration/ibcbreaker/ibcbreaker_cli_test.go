//go:build test
// +build test

package ibcbreaker

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	evmdcmd "github.com/cosmos/evm/evmd/cmd/evmd/cmd"
	cli "github.com/cosmos/evm/evmd/tests/integration/cli"
	network "github.com/cosmos/evm/evmd/tests/network"
	ibcbreakertypes "github.com/cosmos/evm/x/ibcbreaker/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

func TestIbcBreakerCLIDemo(t *testing.T) {
	evmtypes.NewEVMConfigurator().ResetTestConfig()
	t.Cleanup(func() {
		evmtypes.NewEVMConfigurator().ResetTestConfig()
	})

	cfg := network.DefaultConfig()
	evmtypes.NewEVMConfigurator().ResetTestConfig()
	cfg.NumValidators = 1
	cfg.InjectDenomMetadata = true
	cfg.TimeoutCommit = 200 * time.Millisecond
	cfg.APIAddress = "tcp://0.0.0.0:" + cli.FreePort(t)
	cfg.RPCAddress = "tcp://0.0.0.0:" + cli.FreePort(t)
	cfg.GRPCAddress = "0.0.0.0:" + cli.FreePort(t)
	cfg.JSONRPCAddress = "0.0.0.0:" + cli.FreePort(t)

	minSelf, ok := sdkmath.NewIntFromString("888888000000000001000000")
	require.True(t, ok)
	cfg.GentxSelfDelegation = &minSelf
	cfg.BondedTokens = minSelf
	cfg.StakingTokens = minSelf.MulRaw(2)
	cfg.AccountTokens = minSelf.MulRaw(2)

	var govGen govv1.GenesisState
	cfg.Codec.MustUnmarshalJSON(cfg.GenesisState[govtypes.ModuleName], &govGen)
	govGen.Params.MinDeposit = sdk.NewCoins(sdk.NewCoin(cfg.BondDenom, sdkmath.NewInt(1)))
	govGen.Params.ExpeditedMinDeposit = sdk.NewCoins(sdk.NewCoin(cfg.BondDenom, sdkmath.NewInt(1)))
	votingPeriod := 5 * time.Second
	maxDepositPeriod := 5 * time.Second
	govGen.Params.VotingPeriod = &votingPeriod
	govGen.Params.MaxDepositPeriod = &maxDepositPeriod
	cfg.GenesisState[govtypes.ModuleName] = cfg.Codec.MustMarshalJSON(&govGen)

	var ibcBreakerGen ibcbreakertypes.GenesisState
	cfg.Codec.MustUnmarshalJSON(cfg.GenesisState[ibcbreakertypes.ModuleName], &ibcBreakerGen)
	ibcBreakerGen.State.IbcAvailable = true
	ibcBreakerGen.Params.Whitelist = []string{}
	cfg.GenesisState[ibcbreakertypes.ModuleName] = cfg.Codec.MustMarshalJSON(&ibcBreakerGen)

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

	rootCmd := evmdcmd.NewRootCmd()
	feeAmount := "40000000000000000" + cfg.BondDenom
	queryClient := ibcbreakertypes.NewQueryClient(clientCtx1)

	_, err = netw.WaitForHeight(1)
	require.NoError(t, err)

	fundAmount := "2000000000000000000" + cfg.BondDenom
	_ = cli.ExecCLICmd(t, clientCtx1, rootCmd,
		"tx", "bank", "send", val1.Address.String(), addr2.String(), fundAmount,
		"--from", val1.Moniker,
		"--chain-id", cfg.ChainID,
		"--fees", feeAmount,
		"--broadcast-mode", "sync",
		"--gas", "200000",
		"--yes",
	)
	require.NoError(t, netw.WaitForNextBlock())

	cli.SubmitWhitelistProposal(
		t,
		netw,
		rootCmd,
		clientCtx1,
		feeAmount,
		cfg.ChainID,
		"/cosmos.evm.ibcbreaker.v1.MsgUpdateParams",
		"Update ibcbreaker whitelist",
		[]string{val1.Address.String()},
	)

	_ = cli.ExecCLICmd(t, clientCtx1, rootCmd, "query", "ibcbreaker", "whitelist", "--output", "json")
	wlResp, err := queryClient.Whitelist(context.Background(), &ibcbreakertypes.QueryWhitelistRequest{})
	require.NoError(t, err)
	require.Contains(t, wlResp.Whitelist, val1.Address.String())

	_ = cli.ExecCLICmd(t, clientCtx1, rootCmd,
		"tx", "ibcbreaker", "update-ibcbreaker", "false",
		"--from", val1.Moniker,
		"--chain-id", cfg.ChainID,
		"--fees", feeAmount,
		"--broadcast-mode", "sync",
		"--gas", "200000",
		"--yes",
	)
	require.NoError(t, netw.WaitForNextBlock())

	_ = cli.ExecCLICmd(t, clientCtx1, rootCmd, "query", "ibcbreaker", "ibc-available", "--output", "json")
	sysResp, err := queryClient.IbcAvailable(context.Background(), &ibcbreakertypes.QueryIbcAvailableRequest{})
	require.NoError(t, err)
	require.False(t, sysResp.IbcAvailable)

	_ = cli.ExecCLICmd(t, clientCtx1, rootCmd,
		"tx", "ibcbreaker", "update-ibcbreaker", "true",
		"--from", val1.Moniker,
		"--chain-id", cfg.ChainID,
		"--fees", feeAmount,
		"--broadcast-mode", "sync",
		"--gas", "200000",
		"--yes",
	)
	require.NoError(t, netw.WaitForNextBlock())

	_ = cli.ExecCLICmd(t, clientCtx1, rootCmd, "query", "ibcbreaker", "ibc-available", "--output", "json")
	sysResp, err = queryClient.IbcAvailable(context.Background(), &ibcbreakertypes.QueryIbcAvailableRequest{})
	require.NoError(t, err)
	require.True(t, sysResp.IbcAvailable)

	out, err := cli.ExecCLICmdErr(t, clientCtx2, rootCmd,
		"tx", "ibcbreaker", "update-ibcbreaker", "false",
		"--from", "user2",
		"--chain-id", cfg.ChainID,
		"--fees", feeAmount,
		"--broadcast-mode", "sync",
		"--gas", "200000",
		"--yes",
	)
	cli.RequireTxFailed(t, netw, clientCtx2, out, err)

	cli.SubmitWhitelistProposal(
		t,
		netw,
		rootCmd,
		clientCtx1,
		feeAmount,
		cfg.ChainID,
		"/cosmos.evm.ibcbreaker.v1.MsgUpdateParams",
		"Update ibcbreaker whitelist",
		[]string{val1.Address.String(), addr2.String()},
	)

	_ = cli.ExecCLICmd(t, clientCtx1, rootCmd, "query", "ibcbreaker", "whitelist", "--output", "json")
	wlResp, err = queryClient.Whitelist(context.Background(), &ibcbreakertypes.QueryWhitelistRequest{})
	require.NoError(t, err)
	require.Contains(t, wlResp.Whitelist, addr2.String())

	_ = cli.ExecCLICmd(t, clientCtx2, rootCmd,
		"tx", "ibcbreaker", "update-ibcbreaker", "false",
		"--from", "user2",
		"--chain-id", cfg.ChainID,
		"--fees", feeAmount,
		"--broadcast-mode", "sync",
		"--gas", "200000",
		"--yes",
	)
	require.NoError(t, netw.WaitForNextBlock())

	_ = cli.ExecCLICmd(t, clientCtx1, rootCmd, "query", "ibcbreaker", "ibc-available", "--output", "json")
	sysResp, err = queryClient.IbcAvailable(context.Background(), &ibcbreakertypes.QueryIbcAvailableRequest{})
	require.NoError(t, err)
	require.False(t, sysResp.IbcAvailable)

	_ = cli.ExecCLICmd(t, clientCtx2, rootCmd,
		"tx", "ibcbreaker", "update-ibcbreaker", "true",
		"--from", "user2",
		"--chain-id", cfg.ChainID,
		"--fees", feeAmount,
		"--broadcast-mode", "sync",
		"--gas", "200000",
		"--yes",
	)
	require.NoError(t, netw.WaitForNextBlock())

	_ = cli.ExecCLICmd(t, clientCtx1, rootCmd, "query", "ibcbreaker", "ibc-available", "--output", "json")
	sysResp, err = queryClient.IbcAvailable(context.Background(), &ibcbreakertypes.QueryIbcAvailableRequest{})
	require.NoError(t, err)
	require.True(t, sysResp.IbcAvailable)
}
