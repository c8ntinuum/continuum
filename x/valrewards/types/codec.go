package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

var (
	amino = codec.NewLegacyAmino()

	ModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	AminoCdc = codec.NewLegacyAmino()
)

const (
	msgUpdateParamsName       = "cosmos/evm/x/valrewards/MsgUpdateParams"
	msgSetBlocksInEpochName   = "cosmos/evm/x/valrewards/MsgSetBlocksInEpoch"
	msgSetRewardsPerEpochName = "cosmos/evm/x/valrewards/MsgSetRewardsPerEpoch"
	msgSetRewardingPausedName = "cosmos/evm/x/valrewards/MsgSetRewardingPaused"
	msgClaimRewardsName       = "cosmos/evm/x/valrewards/MsgClaimRewards"
	msgDepositRewardsPoolName = "cosmos/evm/x/valrewards/MsgDepositRewardsPool"
)

func init() {
	RegisterLegacyAminoCodec(amino)
	amino.Seal()
}

func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgUpdateParams{},
		&MsgSetBlocksInEpoch{},
		&MsgSetRewardsPerEpoch{},
		&MsgSetRewardingPaused{},
		&MsgClaimRewards{},
		&MsgDepositRewardsPool{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgUpdateParams{}, msgUpdateParamsName, nil)
	cdc.RegisterConcrete(&MsgSetBlocksInEpoch{}, msgSetBlocksInEpochName, nil)
	cdc.RegisterConcrete(&MsgSetRewardsPerEpoch{}, msgSetRewardsPerEpochName, nil)
	cdc.RegisterConcrete(&MsgSetRewardingPaused{}, msgSetRewardingPausedName, nil)
	cdc.RegisterConcrete(&MsgClaimRewards{}, msgClaimRewardsName, nil)
	cdc.RegisterConcrete(&MsgDepositRewardsPool{}, msgDepositRewardsPoolName, nil)
}
