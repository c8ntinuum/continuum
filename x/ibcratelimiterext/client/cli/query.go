package cli

import (
	"context"

	"github.com/spf13/cobra"

	ibcexttypes "github.com/cosmos/evm/x/ibcratelimiterext/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
)

func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        ibcexttypes.ModuleName,
		Short:                      "IBC rate limiter extension query subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetCmdQueryWhitelist(),
	)
	return cmd
}

func GetCmdQueryWhitelist() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "whitelist",
		Short: "Query the operator whitelist managed by ibcratelimiterext",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := ibcexttypes.NewQueryClient(clientCtx)
			res, err := queryClient.Whitelist(context.Background(), &ibcexttypes.QueryWhitelistRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
