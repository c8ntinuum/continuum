package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/evm/evmd"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
)

func genesisCommands(txConfig client.TxConfig, moduleBasics module.BasicManager, appCodec codec.Codec, defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "genesis",
		Short:                      "Application's genesis-related subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	gentxModule := moduleBasics[genutiltypes.ModuleName].(genutil.AppModuleBasic)

	cmd.AddCommand(
		genutilcli.GenTxCmd(moduleBasics, txConfig, banktypes.GenesisBalancesIterator{}, defaultNodeHome, txConfig.SigningContext().ValidatorAddressCodec()),
		genutilcli.MigrateGenesisCmd(genutilcli.MigrationMap),
		genutilcli.CollectGenTxsCmd(banktypes.GenesisBalancesIterator{}, defaultNodeHome, gentxModule.GenTxValidator, txConfig.SigningContext().ValidatorAddressCodec()),
		validateGenesisCmd(moduleBasics, appCodec),
		genutilcli.AddGenesisAccountCmd(defaultNodeHome, txConfig.SigningContext().AddressCodec()),
		genutilcli.AddBulkGenesisAccountCmd(defaultNodeHome, txConfig.SigningContext().AddressCodec()),
	)

	return cmd
}

func validateGenesisCmd(mbm module.BasicManager, appCodec codec.Codec) *cobra.Command {
	return &cobra.Command{
		Use:     "validate [file]",
		Aliases: []string{"validate-genesis"},
		Args:    cobra.RangeArgs(0, 1),
		Short:   "Validates the genesis file at the default location or at the location passed as an arg",
		RunE: func(cmd *cobra.Command, args []string) error {
			serverCtx := server.GetServerContextFromCmd(cmd)
			clientCtx := client.GetClientContextFromCmd(cmd)
			cdc := clientCtx.Codec
			if cdc == nil {
				cdc = appCodec
			}

			genesisFile := serverCtx.Config.GenesisFile()
			if len(args) == 1 {
				genesisFile = args[0]
			}

			appGenesis, err := genutiltypes.AppGenesisFromFile(genesisFile)
			if err != nil {
				return enrichGenesisUnmarshalError(err)
			}

			if err := appGenesis.ValidateAndComplete(); err != nil {
				return fmt.Errorf("make sure that you have correctly migrated all CometBFT consensus params: %w", err)
			}

			var genState evmd.GenesisState
			if err := json.Unmarshal(appGenesis.AppState, &genState); err != nil {
				if strings.Contains(err.Error(), "unexpected end of JSON input") {
					return fmt.Errorf("app_state is missing in the genesis file: %s", err.Error())
				}
				return fmt.Errorf("error unmarshalling genesis doc %s: %w", genesisFile, err)
			}

			if err := validateGenesisModules(mbm, cdc, clientCtx.TxConfig, genState); err != nil {
				return wrapGenesisValidationError(genesisFile, err)
			}

			if err := evmd.CrossGenesisValidateAtInitialHeight(cdc, genState, appGenesis.InitialHeight); err != nil {
				return fmt.Errorf("error validating genesis file %s: %w", genesisFile, err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "File at %s is a valid genesis file\n", genesisFile)
			return nil
		},
	}
}

func validateGenesisModules(mbm module.BasicManager, cdc codec.JSONCodec, txCfg client.TxConfig, genState evmd.GenesisState) error {
	for moduleName, basicMod := range mbm {
		mod, ok := basicMod.(module.HasGenesisBasics)
		if !ok {
			continue
		}

		bz := genState[moduleName]
		switch moduleName {
		case crisistypes.ModuleName:
			if isGenesisSectionMissingOrNull(bz) {
				continue
			}
		case banktypes.ModuleName:
			if err := validateBankGenesisCompat(cdc, bz); err != nil {
				return err
			}
			continue
		}

		if err := mod.ValidateGenesis(cdc, txCfg, bz); err != nil {
			return err
		}
	}

	return nil
}

func validateBankGenesisCompat(cdc codec.JSONCodec, bz json.RawMessage) error {
	var data banktypes.GenesisState
	if err := cdc.UnmarshalJSON(bz, &data); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", banktypes.ModuleName, err)
	}

	if len(data.Params.SendEnabled) > 0 && len(data.SendEnabled) > 0 {
		return errors.New("send_enabled defined in both the send_enabled field and in params (deprecated)")
	}

	if err := data.Params.Validate(); err != nil {
		return err
	}

	seenSendEnabled := make(map[string]bool)
	seenBalances := make(map[string]bool)
	seenMetadatas := make(map[string]bool)
	totalSupply := sdk.Coins{}

	for _, sendEnabled := range data.GetAllSendEnabled() {
		if _, exists := seenSendEnabled[sendEnabled.Denom]; exists {
			return fmt.Errorf("duplicate send enabled found: '%s'", sendEnabled.Denom)
		}
		if err := sendEnabled.Validate(); err != nil {
			return err
		}
		seenSendEnabled[sendEnabled.Denom] = true
	}

	for _, balance := range data.Balances {
		if seenBalances[balance.Address] {
			return fmt.Errorf("duplicate balance for address %s", balance.Address)
		}
		if err := balance.Validate(); err != nil {
			return err
		}

		seenBalances[balance.Address] = true
		totalSupply = totalSupply.Add(balance.Coins...)
	}

	for _, metadata := range data.DenomMetadata {
		if seenMetadatas[metadata.Base] {
			return fmt.Errorf("duplicate client metadata for denom %s", metadata.Base)
		}

		if isCompatibleCTMMetadata(metadata) {
			if err := validateCompatibleCTMMetadata(metadata); err != nil {
				return err
			}
		} else if err := metadata.Validate(); err != nil {
			return err
		}

		seenMetadatas[metadata.Base] = true
	}

	if !data.Supply.Empty() {
		if err := data.Supply.Validate(); err != nil {
			return err
		}
		if !data.Supply.Equal(totalSupply) {
			return fmt.Errorf("genesis supply is incorrect, expected %v, got %v", data.Supply, totalSupply)
		}
	}

	return nil
}

func isGenesisSectionMissingOrNull(bz json.RawMessage) bool {
	return len(bytesTrimSpace(bz)) == 0 || string(bytesTrimSpace(bz)) == "null"
}

func validateCompatibleCTMMetadata(metadata banktypes.Metadata) error {
	if strings.TrimSpace(metadata.Name) == "" {
		return errors.New("name field cannot be blank")
	}
	if strings.TrimSpace(metadata.Symbol) == "" {
		return errors.New("symbol field cannot be blank")
	}
	if err := sdk.ValidateDenom(metadata.Base); err != nil {
		return fmt.Errorf("invalid metadata base denom: %w", err)
	}
	if err := sdk.ValidateDenom(metadata.Display); err != nil {
		return fmt.Errorf("invalid metadata display denom: %w", err)
	}

	unit := metadata.DenomUnits[0]
	if unit == nil {
		return errors.New("invalid compatible ctm denomination unit: nil")
	}
	if unit.Denom != metadata.Base {
		return fmt.Errorf("metadata's first denomination unit must be the one with base denom '%s'", metadata.Base)
	}
	if unit.Exponent != 18 {
		return fmt.Errorf("the exponent for compatible ctm denomination unit %s must be 18", metadata.Base)
	}

	return unit.Validate()
}

func isCompatibleCTMMetadata(metadata banktypes.Metadata) bool {
	if len(metadata.DenomUnits) != 1 || metadata.DenomUnits[0] == nil {
		return false
	}

	return metadata.Base == "ctm" &&
		metadata.Display == "ctm" &&
		metadata.DenomUnits[0].Denom == "ctm" &&
		metadata.DenomUnits[0].Exponent == 18
}

func bytesTrimSpace(bz json.RawMessage) []byte {
	return []byte(strings.TrimSpace(string(bz)))
}

func wrapGenesisValidationError(genesisFile string, err error) error {
	errStr := fmt.Sprintf("error validating genesis file %s: %s", genesisFile, err.Error())
	if errors.Is(err, io.EOF) {
		errStr = fmt.Sprintf("%s: section is missing in the app_state", errStr)
	}
	return fmt.Errorf("%s", errStr)
}

func enrichGenesisUnmarshalError(err error) error {
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return fmt.Errorf("error at offset %d: %s", syntaxErr.Offset, syntaxErr.Error())
	}
	return err
}
