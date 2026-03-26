//go:build test
// +build test

package ibcbreaker

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/evm/evmd"
	testutil "github.com/cosmos/evm/tests/integration/testutil"
	"github.com/cosmos/evm/testutil/integration/evm/network"
	"github.com/cosmos/evm/testutil/keyring"
	ibcbreakerkeeper "github.com/cosmos/evm/x/ibcbreaker/keeper"
	ibcbreakertypes "github.com/cosmos/evm/x/ibcbreaker/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func IbcBreakerWhitelistGovernance(t *testing.T, create network.CreateEvmApp, options ...network.ConfigOption) {
	kr := keyring.New(2)
	addr1 := kr.GetAccAddr(0)
	addr2 := kr.GetAccAddr(1)

	baseDenom := network.DefaultConfig().GetChainCoins().BaseDenom()
	govGen := testutil.DefaultGovGenesis(baseDenom)

	ibcBreakerGen := &ibcbreakertypes.GenesisState{
		Params: ibcbreakertypes.Params{Whitelist: []string{addr1.String()}},
		State:  ibcbreakertypes.IbcBreakerState{IbcAvailable: true},
	}

	opts := []network.ConfigOption{
		network.WithPreFundedAccounts(kr.GetAllAccAddrs()...),
		network.WithValidatorOperators([]sdk.AccAddress{addr1}),
		network.WithAmountOfValidators(1),
		network.WithCustomGenesis(network.CustomGenesisState{
			ibcbreakertypes.ModuleName: ibcBreakerGen,
			govtypes.ModuleName:        govGen,
		}),
	}
	opts = append(opts, options...)

	nw := network.NewUnitTestNetwork(create, opts...)
	ctx := nw.GetContext()

	app, ok := nw.App.(*evmd.EVMD)
	require.True(t, ok)
	ibcbreakerKeeper := app.IbcBreakerKeeper
	server := ibcbreakerkeeper.NewMsgServerImpl(ibcbreakerKeeper)

	_, err := server.UpdateIbcBreaker(sdk.WrapSDKContext(ctx), &ibcbreakertypes.MsgUpdateIbcBreaker{Signer: addr1.String(), IbcAvailable: false})
	require.NoError(t, err)
	require.False(t, ibcbreakerKeeper.GetIbcAvailable(ctx))

	_, err = server.UpdateIbcBreaker(sdk.WrapSDKContext(ctx), &ibcbreakertypes.MsgUpdateIbcBreaker{Signer: addr1.String(), IbcAvailable: true})
	require.NoError(t, err)
	require.True(t, ibcbreakerKeeper.GetIbcAvailable(ctx))

	_, err = server.UpdateIbcBreaker(sdk.WrapSDKContext(ctx), &ibcbreakertypes.MsgUpdateIbcBreaker{Signer: addr2.String(), IbcAvailable: false})
	require.Error(t, err)

	_, err = server.UpdateParams(sdk.WrapSDKContext(ctx), &ibcbreakertypes.MsgUpdateParams{
		Authority: addr1.String(),
		Params:    &ibcbreakertypes.Params{Whitelist: []string{addr1.String(), addr2.String()}},
	})
	require.Error(t, err)

	govKeeper := nw.App.GetGovKeeper()
	govAuthority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	updateMsg := &ibcbreakertypes.MsgUpdateParams{
		Authority: govAuthority,
		Params:    &ibcbreakertypes.Params{Whitelist: []string{addr1.String(), addr2.String()}},
	}

	ctx = testutil.SubmitAndPassProposal(t, ctx, govKeeper, baseDenom, addr1, updateMsg, "Add whitelist", "add second address")

	params := ibcbreakerKeeper.GetParams(ctx)
	require.Contains(t, params.Whitelist, addr2.String())

	_, err = server.UpdateIbcBreaker(sdk.WrapSDKContext(ctx), &ibcbreakertypes.MsgUpdateIbcBreaker{Signer: addr2.String(), IbcAvailable: false})
	require.NoError(t, err)
	require.False(t, ibcbreakerKeeper.GetIbcAvailable(ctx))

	_, err = server.UpdateIbcBreaker(sdk.WrapSDKContext(ctx), &ibcbreakertypes.MsgUpdateIbcBreaker{Signer: addr2.String(), IbcAvailable: true})
	require.NoError(t, err)
	require.True(t, ibcbreakerKeeper.GetIbcAvailable(ctx))
}

func contains(list []string, value string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}
