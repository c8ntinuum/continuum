//go:build test
// +build test

package circuit

import (
	"context"
	"math/big"
	"strings"
	"testing"
	"time"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	evmdcmd "github.com/cosmos/evm/evmd/cmd/evmd/cmd"
	cli "github.com/cosmos/evm/evmd/tests/integration/cli"
	network "github.com/cosmos/evm/evmd/tests/network"
	circuittype "github.com/cosmos/evm/x/circuit/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

func TestCircuitCLIDemo(t *testing.T) {
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

	var circuitGen circuittype.GenesisState
	cfg.Codec.MustUnmarshalJSON(cfg.GenesisState[circuittype.ModuleName], &circuitGen)
	circuitGen.State.SystemAvailable = true
	circuitGen.Params.Whitelist = []string{}
	cfg.GenesisState[circuittype.ModuleName] = cfg.Codec.MustMarshalJSON(&circuitGen)

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
	queryClient := circuittype.NewQueryClient(clientCtx1)

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

	// Prepare and fund a dedicated EVM sender account for extension-tx route checks.
	evmSenderPrivKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	evmSenderAddr := crypto.PubkeyToAddress(evmSenderPrivKey.PublicKey)
	evmSenderBech32 := sdk.AccAddress(evmSenderAddr.Bytes()).String()
	_ = cli.ExecCLICmd(t, clientCtx1, rootCmd,
		"tx", "bank", "send", val1.Address.String(), evmSenderBech32, fundAmount,
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
		"/cosmos.evm.circuit.v1.MsgUpdateParams",
		"Update circuit whitelist",
		[]string{val1.Address.String()},
	)

	_ = cli.ExecCLICmd(t, clientCtx1, rootCmd, "query", "circuit", "whitelist", "--output", "json")
	wlResp, err := queryClient.Whitelist(context.Background(), &circuittype.QueryWhitelistRequest{})
	require.NoError(t, err)
	require.Contains(t, wlResp.Whitelist, val1.Address.String())

	_ = cli.ExecCLICmd(t, clientCtx1, rootCmd,
		"tx", "circuit", "update-circuit", "false",
		"--from", val1.Moniker,
		"--chain-id", cfg.ChainID,
		"--fees", feeAmount,
		"--broadcast-mode", "sync",
		"--gas", "200000",
		"--yes",
	)
	require.NoError(t, netw.WaitForNextBlock())

	_ = cli.ExecCLICmd(t, clientCtx1, rootCmd, "query", "circuit", "system-available", "--output", "json")
	sysResp, err := queryClient.SystemAvailable(context.Background(), &circuittype.QuerySystemAvailableRequest{})
	require.NoError(t, err)
	require.False(t, sysResp.SystemAvailable)

	// Control-path tx must remain allowed while system is unavailable.
	controlPathOut := cli.ExecCLICmd(t, clientCtx1, rootCmd,
		"tx", "circuit", "update-circuit", "false",
		"--from", val1.Moniker,
		"--chain-id", cfg.ChainID,
		"--fees", feeAmount,
		"--broadcast-mode", "sync",
		"--gas", "200000",
		"--yes",
	)
	controlPathCode := cli.WaitForTxResultCode(t, netw, clientCtx1, cli.ExtractTxHash(t, controlPathOut))
	require.Zero(t, controlPathCode, "expected MsgUpdateCircuit control-path tx to pass while circuit is disabled")

	// While system is unavailable, a normal native Cosmos tx should be rejected by ante.
	blockedSendAmount := "1" + cfg.BondDenom
	out, err := cli.ExecCLICmdErr(t, clientCtx1, rootCmd,
		"tx", "bank", "send", val1.Address.String(), addr2.String(), blockedSendAmount,
		"--from", val1.Moniker,
		"--chain-id", cfg.ChainID,
		"--fees", feeAmount,
		"--broadcast-mode", "sync",
		"--gas", "200000",
		"--yes",
	)
	cli.RequireTxFailed(t, netw, clientCtx1, out, err)

	// While system is unavailable, an EVM extension tx should also be rejected by ante.
	evmChainID, err := val1.JSONRPCClient.ChainID(context.Background())
	require.NoError(t, err)
	nonce, err := val1.JSONRPCClient.PendingNonceAt(context.Background(), evmSenderAddr)
	require.NoError(t, err)
	gasPrice, err := val1.JSONRPCClient.SuggestGasPrice(context.Background())
	require.NoError(t, err)
	if gasPrice.Sign() <= 0 {
		gasPrice = big.NewInt(1)
	}
	gasPrice = new(big.Int).Mul(gasPrice, big.NewInt(2))
	toAddr := evmSenderAddr
	rawEthTx := ethtypes.NewTx(&ethtypes.LegacyTx{
		Nonce:    nonce,
		To:       &toAddr,
		Value:    big.NewInt(1),
		Gas:      21000,
		GasPrice: gasPrice,
	})
	ethSigner := ethtypes.LatestSignerForChainID(evmChainID)
	signedEthTx, err := ethtypes.SignTx(rawEthTx, ethSigner, evmSenderPrivKey)
	require.NoError(t, err)
	msgEthTx := &evmtypes.MsgEthereumTx{}
	require.NoError(t, msgEthTx.FromSignedEthereumTx(signedEthTx, ethSigner))
	evmParamsResp, err := evmtypes.NewQueryClient(clientCtx1).Params(context.Background(), &evmtypes.QueryParamsRequest{})
	require.NoError(t, err)
	cosmosEvmTx, err := msgEthTx.BuildTxWithEvmParams(clientCtx1.TxConfig.NewTxBuilder(), evmParamsResp.Params)
	require.NoError(t, err)
	cosmosEvmTxBz, err := clientCtx1.TxConfig.TxEncoder()(cosmosEvmTx)
	require.NoError(t, err)
	evmBroadcastRes, err := val1.RPCClient.BroadcastTxSync(context.Background(), cosmosEvmTxBz)
	require.NoError(t, err)
	require.NotZero(t, evmBroadcastRes.Code, "expected EVM extension tx rejection while circuit is disabled")
	require.Contains(t, strings.ToLower(evmBroadcastRes.Log), "system unavailable")

	// While system is unavailable, JSON-RPC eth_sendRawTransaction should also be rejected.
	jsonrpcEthTx := ethtypes.NewTx(&ethtypes.LegacyTx{
		Nonce:    nonce,
		To:       &toAddr,
		Value:    big.NewInt(2),
		Gas:      21000,
		GasPrice: gasPrice,
	})
	jsonrpcSignedEthTx, err := ethtypes.SignTx(jsonrpcEthTx, ethSigner, evmSenderPrivKey)
	require.NoError(t, err)
	err = val1.JSONRPCClient.SendTransaction(context.Background(), jsonrpcSignedEthTx)
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "system unavailable")

	_ = cli.ExecCLICmd(t, clientCtx1, rootCmd,
		"tx", "circuit", "update-circuit", "true",
		"--from", val1.Moniker,
		"--chain-id", cfg.ChainID,
		"--fees", feeAmount,
		"--broadcast-mode", "sync",
		"--gas", "200000",
		"--yes",
	)
	require.NoError(t, netw.WaitForNextBlock())

	_ = cli.ExecCLICmd(t, clientCtx1, rootCmd, "query", "circuit", "system-available", "--output", "json")
	sysResp, err = queryClient.SystemAvailable(context.Background(), &circuittype.QuerySystemAvailableRequest{})
	require.NoError(t, err)
	require.True(t, sysResp.SystemAvailable)

	out, err = cli.ExecCLICmdErr(t, clientCtx2, rootCmd,
		"tx", "circuit", "update-circuit", "false",
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
		"/cosmos.evm.circuit.v1.MsgUpdateParams",
		"Update circuit whitelist",
		[]string{val1.Address.String(), addr2.String()},
	)

	_ = cli.ExecCLICmd(t, clientCtx1, rootCmd, "query", "circuit", "whitelist", "--output", "json")
	wlResp, err = queryClient.Whitelist(context.Background(), &circuittype.QueryWhitelistRequest{})
	require.NoError(t, err)
	require.Contains(t, wlResp.Whitelist, addr2.String())

	_ = cli.ExecCLICmd(t, clientCtx2, rootCmd,
		"tx", "circuit", "update-circuit", "false",
		"--from", "user2",
		"--chain-id", cfg.ChainID,
		"--fees", feeAmount,
		"--broadcast-mode", "sync",
		"--gas", "200000",
		"--yes",
	)
	require.NoError(t, netw.WaitForNextBlock())

	_ = cli.ExecCLICmd(t, clientCtx1, rootCmd, "query", "circuit", "system-available", "--output", "json")
	sysResp, err = queryClient.SystemAvailable(context.Background(), &circuittype.QuerySystemAvailableRequest{})
	require.NoError(t, err)
	require.False(t, sysResp.SystemAvailable)

	_ = cli.ExecCLICmd(t, clientCtx2, rootCmd,
		"tx", "circuit", "update-circuit", "true",
		"--from", "user2",
		"--chain-id", cfg.ChainID,
		"--fees", feeAmount,
		"--broadcast-mode", "sync",
		"--gas", "200000",
		"--yes",
	)
	require.NoError(t, netw.WaitForNextBlock())

	_ = cli.ExecCLICmd(t, clientCtx1, rootCmd, "query", "circuit", "system-available", "--output", "json")
	sysResp, err = queryClient.SystemAvailable(context.Background(), &circuittype.QuerySystemAvailableRequest{})
	require.NoError(t, err)
	require.True(t, sysResp.SystemAvailable)
}
