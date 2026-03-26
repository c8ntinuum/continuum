//go:build test
// +build test

package circuit

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/evm/evmd"
	testutil "github.com/cosmos/evm/tests/integration/testutil"
	"github.com/cosmos/evm/testutil/integration/evm/network"
	"github.com/cosmos/evm/testutil/keyring"
	circuitkeeper "github.com/cosmos/evm/x/circuit/keeper"
	circuittype "github.com/cosmos/evm/x/circuit/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func CircuitWhitelistGovernance(t *testing.T, create network.CreateEvmApp, options ...network.ConfigOption) {
	kr := keyring.New(2)
	addr1 := kr.GetAccAddr(0)
	addr2 := kr.GetAccAddr(1)

	baseDenom := network.DefaultConfig().GetChainCoins().BaseDenom()
	govGen := testutil.DefaultGovGenesis(baseDenom)

	circuitGen := &circuittype.GenesisState{
		Params: circuittype.Params{Whitelist: []string{addr1.String()}},
		State:  circuittype.CircuitState{SystemAvailable: true},
	}

	opts := []network.ConfigOption{
		network.WithPreFundedAccounts(kr.GetAllAccAddrs()...),
		network.WithValidatorOperators([]sdk.AccAddress{addr1}),
		network.WithAmountOfValidators(1),
		network.WithCustomGenesis(network.CustomGenesisState{
			circuittype.ModuleName: circuitGen,
			govtypes.ModuleName:    govGen,
		}),
	}
	opts = append(opts, options...)

	nw := network.NewUnitTestNetwork(create, opts...)
	ctx := nw.GetContext()

	app, ok := nw.App.(*evmd.EVMD)
	require.True(t, ok)
	circuitKeeper := app.CircuitKeeper
	server := circuitkeeper.NewMsgServerImpl(circuitKeeper)

	_, err := server.UpdateCircuit(sdk.WrapSDKContext(ctx), &circuittype.MsgUpdateCircuit{Signer: addr1.String(), SystemAvailable: false})
	require.NoError(t, err)
	require.False(t, circuitKeeper.GetSystemAvailable(ctx))

	_, err = server.UpdateCircuit(sdk.WrapSDKContext(ctx), &circuittype.MsgUpdateCircuit{Signer: addr1.String(), SystemAvailable: true})
	require.NoError(t, err)
	require.True(t, circuitKeeper.GetSystemAvailable(ctx))

	_, err = server.UpdateCircuit(sdk.WrapSDKContext(ctx), &circuittype.MsgUpdateCircuit{Signer: addr2.String(), SystemAvailable: false})
	require.Error(t, err)

	govKeeper := nw.App.GetGovKeeper()
	govAuthority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	updateMsg := &circuittype.MsgUpdateParams{
		Authority: govAuthority,
		Params:    &circuittype.Params{Whitelist: []string{addr1.String(), addr2.String()}},
	}

	ctx = testutil.SubmitAndPassProposal(t, ctx, govKeeper, baseDenom, addr1, updateMsg, "Add whitelist", "add second address")

	params := circuitKeeper.GetParams(ctx)
	require.Contains(t, params.Whitelist, addr2.String())

	_, err = server.UpdateCircuit(sdk.WrapSDKContext(ctx), &circuittype.MsgUpdateCircuit{Signer: addr2.String(), SystemAvailable: false})
	require.NoError(t, err)
	require.False(t, circuitKeeper.GetSystemAvailable(ctx))

	_, err = server.UpdateCircuit(sdk.WrapSDKContext(ctx), &circuittype.MsgUpdateCircuit{Signer: addr2.String(), SystemAvailable: true})
	require.NoError(t, err)
	require.True(t, circuitKeeper.GetSystemAvailable(ctx))
}

func contains(list []string, value string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}
