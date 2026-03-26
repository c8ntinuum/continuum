package cli

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/evm/x/ibcbreaker/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
)

func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "ibcbreaker subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewUpdateIbcBreakerCmd(),
	)
	return txCmd
}

func NewUpdateIbcBreakerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-ibcbreaker [IBC_AVAILABLE]",
		Short: "Update the IBC availability flag",
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

			msg := &types.MsgUpdateIbcBreaker{
				Signer:       clientCtx.GetFromAddress().String(),
				IbcAvailable: available,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
