package addresstable

import (
	"fmt"
	"math/big"
	"slices"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	evmconfig "github.com/cosmos/evm/config"
	addressprecompile "github.com/cosmos/evm/precompiles/addresstable"
	"github.com/cosmos/evm/testutil/integration/evm/factory"
	"github.com/cosmos/evm/testutil/integration/evm/grpc"
	"github.com/cosmos/evm/testutil/integration/evm/network"
	testkeyring "github.com/cosmos/evm/testutil/keyring"
	testutiltypes "github.com/cosmos/evm/testutil/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

const (
	methodAddressExists = "addressExists"
	methodRegister      = "register"
)

type IntegrationTestSuite struct {
	network        *network.UnitTestNetwork
	factory        factory.TxFactory
	keyring        testkeyring.Keyring
	precompileAddr common.Address
	precompileABI  abi.ABI
}

func NewIntegrationTestSuite(create network.CreateEvmApp, options ...network.ConfigOption) *IntegrationTestSuite {
	keyring := testkeyring.New(2)

	opts := []network.ConfigOption{
		network.WithPreFundedAccounts(keyring.GetAllAccAddrs()...),
		network.WithOtherDenoms([]string{evmconfig.ExampleChainDenom}),
	}
	opts = append(opts, options...)

	integrationNetwork := network.NewUnitTestNetwork(create, opts...)
	grpcHandler := grpc.NewIntegrationHandler(integrationNetwork)
	txFactory := factory.New(integrationNetwork, grpcHandler)

	precompile, err := addressprecompile.NewPrecompile(integrationNetwork.App.GetBankKeeper())
	if err != nil {
		panic(fmt.Errorf("failed to instantiate addresstable precompile: %w", err))
	}

	return &IntegrationTestSuite{
		network:        integrationNetwork,
		factory:        txFactory,
		keyring:        keyring,
		precompileAddr: precompile.Address(),
		precompileABI:  precompile.ABI,
	}
}

func (s *IntegrationTestSuite) registerAddress(addr common.Address) (*big.Int, error) {
	txArgs := evmtypes.EvmTxArgs{To: &s.precompileAddr}
	callArgs := testutiltypes.CallArgs{
		ContractABI: s.precompileABI,
		MethodName:  methodRegister,
		Args:        []interface{}{addr},
	}

	txResult, err := s.factory.ExecuteContractCall(s.keyring.GetPrivKey(0), txArgs, callArgs)
	if err != nil {
		return nil, err
	}

	ethRes, err := evmtypes.DecodeTxResponse(txResult.Data)
	if err != nil {
		return nil, err
	}

	out, err := s.precompileABI.Unpack(methodRegister, ethRes.Ret)
	if err != nil {
		return nil, err
	}
	idx, ok := out[0].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("unexpected register output type: %T", out[0])
	}

	return idx, nil
}

func (s *IntegrationTestSuite) addressExists(addr common.Address) (bool, error) {
	queryArgs := evmtypes.EvmTxArgs{To: &s.precompileAddr}
	callArgs := testutiltypes.CallArgs{
		ContractABI: s.precompileABI,
		MethodName:  methodAddressExists,
		Args:        []interface{}{addr},
	}

	ethRes, err := s.factory.QueryContract(queryArgs, callArgs, 0)
	if err != nil {
		return false, err
	}

	out, err := s.precompileABI.Unpack(methodAddressExists, ethRes.Ret)
	if err != nil {
		return false, err
	}
	exists, ok := out[0].(bool)
	if !ok {
		return false, fmt.Errorf("unexpected addressExists output type: %T", out[0])
	}

	return exists, nil
}

func (s *IntegrationTestSuite) setAddressTableEnabledByGov(t *testing.T, enabled bool) {
	t.Helper()

	ctx := s.network.GetContext()
	params := s.network.App.GetEVMKeeper().GetParams(ctx)
	addr := strings.ToLower(s.precompileAddr.Hex())

	activePrecompiles := make([]string, 0, len(params.ActiveStaticPrecompiles))
	for _, p := range params.ActiveStaticPrecompiles {
		activePrecompiles = append(activePrecompiles, strings.ToLower(p))
	}
	hasAddr := slices.Contains(activePrecompiles, addr)
	if enabled == hasAddr {
		return
	}

	if enabled {
		params.ActiveStaticPrecompiles = append(params.ActiveStaticPrecompiles, addr)
	} else {
		filtered := make([]string, 0, len(params.ActiveStaticPrecompiles))
		for _, precompileAddr := range params.ActiveStaticPrecompiles {
			if strings.ToLower(precompileAddr) != addr {
				filtered = append(filtered, precompileAddr)
			}
		}
		params.ActiveStaticPrecompiles = filtered
	}

	updateMsg := &evmtypes.MsgUpdateParams{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Params:    params,
	}

	_, err := s.network.App.GetEVMKeeper().UpdateParams(sdk.WrapSDKContext(ctx), updateMsg)
	require.NoError(t, err)

	updated := s.network.App.GetEVMKeeper().GetParams(ctx)
	updatedActive := make([]string, 0, len(updated.ActiveStaticPrecompiles))
	for _, p := range updated.ActiveStaticPrecompiles {
		updatedActive = append(updatedActive, strings.ToLower(p))
	}
	require.Equal(t, enabled, slices.Contains(updatedActive, addr))
}

// TestAddressTablePrecompileIntegrationTestSuite covers:
// 1. register(address) followed by addressExists(address).
// 2. register -> disable by governance -> register/read fail -> enable by governance -> register/read succeed.
func TestAddressTablePrecompileIntegrationTestSuite(t *testing.T, create network.CreateEvmApp, options ...network.ConfigOption) {
	t.Run("register_and_address_exists", func(t *testing.T) {
		s := NewIntegrationTestSuite(create, options...)
		addr := common.HexToAddress("0x1111111111111111111111111111111111111111")

		idx, err := s.registerAddress(addr)
		require.NoError(t, err)
		require.Equal(t, uint64(0), idx.Uint64())
		require.NoError(t, s.network.NextBlock())

		exists, err := s.addressExists(addr)
		require.NoError(t, err)
		require.True(t, exists)
	})

	t.Run("toggle_by_governance_and_restore", func(t *testing.T) {
		s := NewIntegrationTestSuite(create, options...)

		firstAddr := common.HexToAddress("0x2222222222222222222222222222222222222222")
		secondAddr := common.HexToAddress("0x3333333333333333333333333333333333333333")

		firstIdx, err := s.registerAddress(firstAddr)
		require.NoError(t, err)
		require.Equal(t, uint64(0), firstIdx.Uint64())
		require.NoError(t, s.network.NextBlock())

		firstExists, err := s.addressExists(firstAddr)
		require.NoError(t, err)
		require.True(t, firstExists)

		s.setAddressTableEnabledByGov(t, false)

		_, err = s.registerAddress(secondAddr)
		require.Error(t, err, "register must fail while addresstable precompile is disabled")
		require.NoError(t, s.network.NextBlock())

		firstExistsWhileDisabled, err := s.addressExists(firstAddr)
		require.NoError(t, err)
		require.True(t, firstExistsWhileDisabled)

		s.setAddressTableEnabledByGov(t, true)

		secondIdx, err := s.registerAddress(secondAddr)
		require.NoError(t, err)
		require.Equal(t, uint64(1), secondIdx.Uint64())
		require.NoError(t, s.network.NextBlock())

		secondExists, err := s.addressExists(secondAddr)
		require.NoError(t, err)
		require.True(t, secondExists)
	})
}
