package valrewards

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	vrtypes "github.com/cosmos/evm/x/valrewards/types"
)

func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: vrtypes.Query_serviceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "DelegationRewards",
					Use:       "delegation-rewards [delegator] [epoch]",
					Short:     "Query delegation rewards for a delegator at an epoch",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "delegator"},
						{ProtoField: "epoch"},
					},
				},
				{
					RpcMethod: "ValidatorOutstandingRewards",
					Use:       "validator-outstanding-rewards [epoch] [validator-address]",
					Short:     "Query validator outstanding rewards at an epoch",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "epoch"},
						{ProtoField: "validator_address"},
					},
				},
				{
					RpcMethod: "RewardsPool",
					Use:       "rewards-pool",
					Short:     "Query the rewards module pool balance",
				},
			},
		},
	}
}
