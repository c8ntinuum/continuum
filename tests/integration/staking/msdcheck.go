package staking

import (
	"testing"

	"github.com/stretchr/testify/require"

	abcitypes "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/evm/config"
	basefactory "github.com/cosmos/evm/testutil/integration/base/factory"
	"github.com/cosmos/evm/testutil/integration/evm/factory"
	"github.com/cosmos/evm/testutil/integration/evm/grpc"
	"github.com/cosmos/evm/testutil/integration/evm/network"
	testkeyring "github.com/cosmos/evm/testutil/keyring"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

const minSelfDelegationStr = "888888000000000000000000"

func TestMSDCheckIntegration(t *testing.T, create network.CreateEvmApp, options ...network.ConfigOption) {
	t.Run("MsgCreateValidator below min fails", func(t *testing.T) {
		nw, tf, kr := setupNetwork(t, create, options...)
		_ = nw

		minSelfDelegation := mustInt(t, minSelfDelegationStr)
		coin := sdk.NewCoin(nw.GetBaseDenom(), minSelfDelegation.SubRaw(1))

		msg := buildCreateValidatorMsg(t, kr.GetPrivKey(0), coin, minSelfDelegation)
		res, err := executeCosmosTx(tf, kr.GetPrivKey(0), msg)
		require.NoError(t, err)
		require.NotZero(t, res.Code)
	})

	t.Run("MsgCreateValidator exact min succeeds", func(t *testing.T) {
		nw, tf, kr := setupNetwork(t, create, options...)
		_ = nw

		minSelfDelegation := mustInt(t, minSelfDelegationStr)
		coin := sdk.NewCoin(nw.GetBaseDenom(), minSelfDelegation)

		msg := buildCreateValidatorMsg(t, kr.GetPrivKey(0), coin, minSelfDelegation)
		res, err := commitCosmosTx(tf, kr.GetPrivKey(0), msg)
		require.NoError(t, err)
		require.Zerof(t, res.Code, "log: %s", res.Log)
	})

	t.Run("MsgCreateValidator above min succeeds", func(t *testing.T) {
		nw, tf, kr := setupNetwork(t, create, options...)
		_ = nw

		minSelfDelegation := mustInt(t, minSelfDelegationStr)
		coin := sdk.NewCoin(nw.GetBaseDenom(), minSelfDelegation.AddRaw(1))

		msg := buildCreateValidatorMsg(t, kr.GetPrivKey(0), coin, minSelfDelegation)
		res, err := commitCosmosTx(tf, kr.GetPrivKey(0), msg)
		require.NoError(t, err)
		require.Zerof(t, res.Code, "log: %s", res.Log)
	})

	t.Run("MsgEditValidator below min stake fails", func(t *testing.T) {
		nw, tf, kr := setupNetwork(t, create, options...)
		_ = nw

		minSelfDelegation := mustInt(t, minSelfDelegationStr)
		coin := sdk.NewCoin(nw.GetBaseDenom(), minSelfDelegation)

		msg := buildCreateValidatorMsg(t, kr.GetPrivKey(0), coin, minSelfDelegation)
		res, err := commitCosmosTx(tf, kr.GetPrivKey(0), msg)
		require.NoError(t, err)
		require.Zerof(t, res.Code, "log: %s", res.Log)

		valAddr := sdk.ValAddress(kr.GetPrivKey(0).PubKey().Address())
		newMin := minSelfDelegation.SubRaw(1)
		editMsg := stakingtypes.NewMsgEditValidator(
			valAddr.String(),
			stakingtypes.NewDescription("moniker", "identity", "", "", ""),
			nil,
			&newMin,
		)

		res, err = executeCosmosTx(tf, kr.GetPrivKey(0), editMsg)
		require.NoError(t, err)
		require.NotZero(t, res.Code)
	})
}

func setupNetwork(t *testing.T, create network.CreateEvmApp, options ...network.ConfigOption) (*network.UnitTestNetwork, factory.TxFactory, testkeyring.Keyring) {
	t.Helper()
	kr := testkeyring.New(2)
	minSelfDelegation := mustInt(t, minSelfDelegationStr)
	opts := []network.ConfigOption{
		network.WithPreFundedAccounts(kr.GetAllAccAddrs()...),
		network.WithBaseCoin(config.ExampleChainDenom, uint8(config.BaseDenomUnit)),
	}
	baseDenom := config.ExampleChainDenom
	balances := make([]banktypes.Balance, 0, len(kr.GetAllAccAddrs()))
	for _, addr := range kr.GetAllAccAddrs() {
		balances = append(balances, banktypes.Balance{
			Address: addr.String(),
			Coins:   sdk.NewCoins(sdk.NewCoin(baseDenom, minSelfDelegation.MulRaw(2))),
		})
	}
	opts = append(opts, network.WithBalances(balances...))
	opts = append(opts, options...)
	nw := network.NewUnitTestNetwork(create, opts...)
	grpcHandler := grpc.NewIntegrationHandler(nw)
	txFactory := factory.New(nw, grpcHandler)
	return nw, txFactory, kr
}

func mustInt(t *testing.T, v string) sdkmath.Int {
	t.Helper()
	i, ok := sdkmath.NewIntFromString(v)
	require.True(t, ok)
	return i
}

func buildCreateValidatorMsg(t *testing.T, priv cryptotypes.PrivKey, selfDelegation sdk.Coin, minSelfDelegation sdkmath.Int) *stakingtypes.MsgCreateValidator {
	t.Helper()
	valAddr := sdk.ValAddress(priv.PubKey().Address()).String()
	consKey := ed25519.GenPrivKey().PubKey()
	desc := stakingtypes.NewDescription("moniker", "identity", "", "", "")
	commission := stakingtypes.NewCommissionRates(
		sdkmath.LegacyNewDecWithPrec(1, 1),
		sdkmath.LegacyNewDecWithPrec(1, 1),
		sdkmath.LegacyNewDecWithPrec(1, 1),
	)

	msg, err := stakingtypes.NewMsgCreateValidator(
		valAddr,
		consKey,
		selfDelegation,
		desc,
		commission,
		minSelfDelegation,
	)
	require.NoError(t, err)
	return msg
}

func executeCosmosTx(tf factory.TxFactory, priv cryptotypes.PrivKey, msgs ...sdk.Msg) (abcitypes.ExecTxResult, error) {
	gas := uint64(600_000)
	gasPrice := sdkmath.NewInt(1_000_000_000)
	return tf.ExecuteCosmosTx(priv, basefactory.CosmosTxArgs{
		Msgs:     msgs,
		Gas:      &gas,
		GasPrice: &gasPrice,
	})
}

func commitCosmosTx(tf factory.TxFactory, priv cryptotypes.PrivKey, msgs ...sdk.Msg) (abcitypes.ExecTxResult, error) {
	gas := uint64(600_000)
	gasPrice := sdkmath.NewInt(1_000_000_000)
	return tf.CommitCosmosTx(priv, basefactory.CosmosTxArgs{
		Msgs:     msgs,
		Gas:      &gas,
		GasPrice: &gasPrice,
	})
}
