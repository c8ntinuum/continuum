package cli

import (
	"strings"

	"github.com/spf13/cobra"

	ibcexttypes "github.com/cosmos/evm/x/ibcratelimiterext/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
)

const FlagWhitelist = "whitelist"

func NewTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        ibcexttypes.ModuleName,
		Short:                      "IBC rate limiter extension transactions",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetTxUpdateParamsCmd(),
	)
	return cmd
}

func GetTxUpdateParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-params",
		Short: "Update ibcratelimiterext params (governance authority only)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			whitelistCSV, err := cmd.Flags().GetString(FlagWhitelist)
			if err != nil {
				return err
			}
			whitelist := []string{}
			if strings.TrimSpace(whitelistCSV) != "" {
				for _, v := range strings.Split(whitelistCSV, ",") {
					s := strings.TrimSpace(v)
					if s != "" {
						whitelist = append(whitelist, s)
					}
				}
			}

			msg := &ibcexttypes.MsgUpdateParams{
				Authority: clientCtx.GetFromAddress().String(),
				Params: &ibcexttypes.Params{
					Whitelist: whitelist,
				},
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagWhitelist, "", "comma-separated operator whitelist addresses")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
