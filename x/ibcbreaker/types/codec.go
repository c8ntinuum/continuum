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
	msgUpdateIbcBreakerName = "cosmos/evm/x/ibcbreaker/MsgUpdateIbcBreaker"
	msgUpdateParamsName     = "cosmos/evm/x/ibcbreaker/MsgUpdateParams"
)

func init() {
	RegisterLegacyAminoCodec(amino)
	amino.Seal()
}

func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgUpdateIbcBreaker{},
		&MsgUpdateParams{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgUpdateIbcBreaker{}, msgUpdateIbcBreakerName, nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, msgUpdateParamsName, nil)
}
