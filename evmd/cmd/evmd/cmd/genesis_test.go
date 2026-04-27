package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"testing"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/server"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/evm/evmd"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

func TestValidateGenesisCmdRejectsCrossModuleMismatch(t *testing.T) {
	app := newValidateGenesisTestApp(t)
	genesisState := app.DefaultGenesis()

	var bankGen banktypes.GenesisState
	app.AppCodec().MustUnmarshalJSON(genesisState[banktypes.ModuleName], &bankGen)
	bankGen.Balances = append(bankGen.Balances, banktypes.Balance{
		Address: sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address().Bytes()).String(),
		Coins:   sdk.NewCoins(sdk.NewCoin("atest", sdkmath.OneInt())),
	})
	genesisState[banktypes.ModuleName] = app.AppCodec().MustMarshalJSON(&bankGen)

	err := runValidateGenesisCmd(t, app, genesisState)
	require.ErrorContains(t, err, "bank balance address")
}

func TestValidateGenesisCmdAllowsMissingCrisisGenesis(t *testing.T) {
	app := newValidateGenesisTestApp(t)
	genesisState := app.DefaultGenesis()
	delete(genesisState, crisistypes.ModuleName)

	err := runValidateGenesisCmd(t, app, genesisState)
	require.NoError(t, err)
}

func TestValidateGenesisCmdRejectsInvalidPresentCrisisGenesis(t *testing.T) {
	app := newValidateGenesisTestApp(t)
	genesisState := app.DefaultGenesis()
	genesisState[crisistypes.ModuleName] = json.RawMessage(`{"constant_fee":{"denom":"ctm","amount":"0"}}`)

	err := runValidateGenesisCmd(t, app, genesisState)
	require.ErrorContains(t, err, "constant fee must be positive")
}

func TestValidateGenesisCmdAllowsCompatibleCTMBankMetadata(t *testing.T) {
	app := newValidateGenesisTestApp(t)
	genesisState := app.DefaultGenesis()

	var bankGen banktypes.GenesisState
	app.AppCodec().MustUnmarshalJSON(genesisState[banktypes.ModuleName], &bankGen)
	bankGen.DenomMetadata = []banktypes.Metadata{compatibleCTMMetadata(18)}
	genesisState[banktypes.ModuleName] = app.AppCodec().MustMarshalJSON(&bankGen)

	err := runValidateGenesisCmd(t, app, genesisState)
	require.NoError(t, err)
}

func TestValidateGenesisCmdRejectsOtherInvalidBankMetadata(t *testing.T) {
	app := newValidateGenesisTestApp(t)
	genesisState := app.DefaultGenesis()

	var bankGen banktypes.GenesisState
	app.AppCodec().MustUnmarshalJSON(genesisState[banktypes.ModuleName], &bankGen)
	bankGen.DenomMetadata = []banktypes.Metadata{compatibleCTMMetadata(17)}
	genesisState[banktypes.ModuleName] = app.AppCodec().MustMarshalJSON(&bankGen)

	err := runValidateGenesisCmd(t, app, genesisState)
	require.ErrorContains(t, err, "the exponent for base denomination unit ctm must be 0")
}

func newValidateGenesisTestApp(t *testing.T) *evmd.EVMD {
	app := evmd.NewExampleApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		true,
		simtestutil.EmptyAppOptions{},
	)

	return app
}

func runValidateGenesisCmd(t *testing.T, app *evmd.EVMD, genesisState map[string]json.RawMessage) error {
	t.Helper()
	normalizeVMGenesisForValidation(t, app, genesisState)

	appState, err := json.Marshal(genesisState)
	require.NoError(t, err)

	genesisFile := filepath.Join(t.TempDir(), "genesis.json")
	require.NoError(t, genutiltypes.NewAppGenesisWithVersion("cross-genesis-cmd-test", appState).SaveAs(genesisFile))

	cmd := validateGenesisCmd(app.BasicModuleManager, app.AppCodec())
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetContext(context.WithValue(
		context.WithValue(context.Background(), client.ClientContextKey, &client.Context{}),
		server.ServerContextKey,
		server.NewDefaultContext(),
	))

	clientCtx := client.Context{}.
		WithCodec(app.AppCodec()).
		WithTxConfig(app.GetTxConfig())
	require.NoError(t, client.SetCmdClientContextHandler(clientCtx, cmd))

	cmd.SetArgs([]string{genesisFile})

	return cmd.Execute()
}

func normalizeVMGenesisForValidation(t *testing.T, app *evmd.EVMD, genesisState map[string]json.RawMessage) {
	t.Helper()

	var vmGen evmtypes.GenesisState
	app.AppCodec().MustUnmarshalJSON(genesisState[evmtypes.ModuleName], &vmGen)
	sort.Strings(vmGen.Params.ActiveStaticPrecompiles)
	genesisState[evmtypes.ModuleName] = app.AppCodec().MustMarshalJSON(&vmGen)
}

func compatibleCTMMetadata(exponent uint32) banktypes.Metadata {
	return banktypes.Metadata{
		Description: "Compatible ctm metadata for validator tests",
		Base:        "ctm",
		Display:     "ctm",
		Name:        fmt.Sprintf("CTM exponent %d", exponent),
		Symbol:      "CTM",
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    "ctm",
				Exponent: exponent,
			},
		},
	}
}
