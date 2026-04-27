package evmd

import (
	"testing"

	"github.com/stretchr/testify/require"

	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	authz "github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/evm/config"
	circuittype "github.com/cosmos/evm/x/circuit/types"
	erc20types "github.com/cosmos/evm/x/erc20/types"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	ibcbreakertypes "github.com/cosmos/evm/x/ibcbreaker/types"
	ibcratelimiterexttypes "github.com/cosmos/evm/x/ibcratelimiterext/types"
	precisebanktypes "github.com/cosmos/evm/x/precisebank/types"
	valrewardstypes "github.com/cosmos/evm/x/valrewards/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
)

func TestInitGenesisOrder(t *testing.T) {
	app, _ := setup(false, 5, "init-order-test", config.EVMChainID)

	expected := []string{
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		stakingtypes.ModuleName,
		slashingtypes.ModuleName,
		govtypes.ModuleName,
		minttypes.ModuleName,
		ibcexported.ModuleName,
		evmtypes.ModuleName,
		feemarkettypes.ModuleName,
		erc20types.ModuleName,
		precisebanktypes.ModuleName,
		valrewardstypes.ModuleName,
		circuittype.ModuleName,
		ibcbreakertypes.ModuleName,
		ibcratelimiterexttypes.ModuleName,
		ibctransfertypes.ModuleName,
	}
	expected = append(expected, optionalRateLimitGenesisModules()...)
	expected = append(expected,
		genutiltypes.ModuleName,
		evidencetypes.ModuleName,
		authz.ModuleName,
		feegrant.ModuleName,
		upgradetypes.ModuleName,
		vestingtypes.ModuleName,
		crisistypes.ModuleName,
	)

	require.Equal(t, expected, app.ModuleManager.OrderInitGenesis)
}
