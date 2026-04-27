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
		NewSetBlocksInEpochCmd(),
		NewSetRewardsPerEpochCmd(),
		NewSetRewardingPausedCmd(),
		NewDepositRewardsPoolCmd(),
		NewClaimRewardsCmd(),
	)

	return txCmd
}
func NewDepositRewardsPoolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deposit [amount]",
		Short: "Deposit funds into the validator rewards pool",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			depositor := cliCtx.GetFromAddress().String()

			coin, err := sdk.ParseCoinNormalized(args[0])
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

func NewSetBlocksInEpochCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-blocks-in-epoch [blocks-in-epoch]",
		Short: "Stage a new blocks_in_epoch value for the next epoch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			blocksInEpoch, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid blocks-in-epoch: %w", err)
			}

			msg := &vrtypes.MsgSetBlocksInEpoch{
				Signer:        clientCtx.GetFromAddress().String(),
				BlocksInEpoch: blocksInEpoch,
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewSetRewardsPerEpochCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-rewards-per-epoch [rewards-per-epoch]",
		Short: "Stage a new rewards_per_epoch value for the next epoch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &vrtypes.MsgSetRewardsPerEpoch{
				Signer:          clientCtx.GetFromAddress().String(),
				RewardsPerEpoch: args[0],
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewSetRewardingPausedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-rewarding-paused [rewarding-paused]",
		Short: "Stage a new rewarding_paused value for the next epoch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			rewardingPaused, err := strconv.ParseBool(args[0])
			if err != nil {
				return fmt.Errorf("invalid rewarding-paused: %w", err)
			}

			msg := &vrtypes.MsgSetRewardingPaused{
				Signer:          clientCtx.GetFromAddress().String(),
				RewardingPaused: rewardingPaused,
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewClaimRewardsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim [validator-address] [epoch]",
		Short: "Trigger a validator rewards claim for an epoch",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			epoch, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid epoch: %w", err)
			}

			msg := &vrtypes.MsgClaimRewards{
				ValidatorOperator: args[0],
				Epoch:             epoch,
				Requester:         cliCtx.GetFromAddress().String(),
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
