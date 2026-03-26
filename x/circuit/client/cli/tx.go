package cli

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/evm/x/circuit/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
)

func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "circuit subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewUpdateCircuitCmd(),
	)
	return txCmd
}

func NewUpdateCircuitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-circuit [SYSTEM_AVAILABLE]",
		Short: "Update the system availability flag",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			available, err := strconv.ParseBool(args[0])
			if err != nil {
				return err
			}

			msg := &types.MsgUpdateCircuit{
				Signer:          clientCtx.GetFromAddress().String(),
				SystemAvailable: available,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
