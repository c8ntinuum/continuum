package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/evm/x/ibcbreaker/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
)

func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the ibcbreaker module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetIbcAvailableCmd(),
		GetWhitelistCmd(),
	)
	return cmd
}

func GetIbcAvailableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ibc-available",
		Short: "Get the IBC availability flag",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.IbcAvailable(cmd.Context(), &types.QueryIbcAvailableRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func GetWhitelistCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "whitelist",
		Short: "Get the ibcbreaker whitelist",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Whitelist(cmd.Context(), &types.QueryWhitelistRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
