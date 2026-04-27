package valrewards

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/evm/testutil/integration/evm/factory"
	"github.com/cosmos/evm/testutil/integration/evm/grpc"
	"github.com/cosmos/evm/testutil/integration/evm/network"
	"github.com/cosmos/evm/testutil/keyring"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

// IntegrationTestSuite is the implementation of the TestSuite interface for valrewards precompile
// integration tests.
type IntegrationTestSuite struct {
	suite.Suite

	create  network.CreateEvmApp
	options []network.ConfigOption

	network     *network.UnitTestNetwork
	factory     factory.TxFactory
	grpcHandler grpc.Handler
	keyring     keyring.Keyring
}

func NewIntegrationTestSuite(create network.CreateEvmApp, options ...network.ConfigOption) *IntegrationTestSuite {
	return &IntegrationTestSuite{
		create:  create,
		options: options,
	}
}

func (is *IntegrationTestSuite) SetupTest() {
	kr := keyring.New(2)
	options := []network.ConfigOption{
		network.WithPreFundedAccounts(kr.GetAllAccAddrs()...),
		network.WithValidatorOperators([]sdktypes.AccAddress{kr.GetAccAddr(0)}),
		network.WithAmountOfValidators(1),
		network.WithOtherDenoms([]string{evmtypes.DefaultEVMDenom}),
	}
	options = append(options, is.options...)

	integrationNetwork := network.NewUnitTestNetwork(is.create, options...)
	grpcHandler := grpc.NewIntegrationHandler(integrationNetwork)
	txFactory := factory.New(integrationNetwork, grpcHandler)

	is.keyring = kr
	is.network = integrationNetwork
	is.factory = txFactory
	is.grpcHandler = grpcHandler
}

func (is *IntegrationTestSuite) precompileAddr() common.Address {
	return common.HexToAddress(evmtypes.ValRewardsPrecompileAddress)
}
