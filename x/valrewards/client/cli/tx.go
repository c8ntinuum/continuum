package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	vrtypes "github.com/cosmos/evm/x/valrewards/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        vrtypes.ModuleName,
		Short:                      "valrewards subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewDepositRewardsPoolCmd(),
		NewClaimRewardsCmd(),
	)

	return txCmd
}
func NewDepositRewardsPoolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deposit [depositor] [amount]",
		Short: "Deposit funds into the validator rewards pool",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			depositor := args[0]
			if _, err := sdk.AccAddressFromBech32(depositor); err != nil {
				return fmt.Errorf("invalid depositor address: %w", err)
			}

			coin, err := sdk.ParseCoinNormalized(args[1])
			if err != nil {
				return err
			}

			msg := &vrtypes.MsgDepositRewardsPool{
				Depositor: depositor,
				Amount:    &coin,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(cliCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
func NewClaimRewardsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim [delegator] [max-retrieve] [epoch]",
		Short: "Claim delegation rewards for an epoch",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			delegator := args[0]
			if _, err := sdk.AccAddressFromBech32(delegator); err != nil {
				return fmt.Errorf("invalid delegator address: %w", err)
			}

			maxRetrieve64, err := strconv.ParseUint(args[1], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid max-retrieve: %w", err)
			}
			maxRetrieve := uint32(maxRetrieve64)

			epoch, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid epoch: %w", err)
			}

			msg := &vrtypes.MsgClaimRewards{
				Delegator:   delegator,
				MaxRetrieve: maxRetrieve,
				Epoch:       epoch,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(cliCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
