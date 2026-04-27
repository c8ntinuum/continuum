package types

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgUpdateIbcBreaker{}
var _ sdk.Msg = &MsgUpdateParams{}

func (m *MsgUpdateIbcBreaker) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Signer); err != nil {
		return errorsmod.Wrap(err, "invalid signer address")
	}
	return nil
}

func (m MsgUpdateIbcBreaker) GetSignBytes() []byte {
	return AminoCdc.MustMarshalJSON(&m)
}

func (m *MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}
	if m.Params == nil {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "params cannot be nil")
	}
	return m.Params.Validate()
}

func (m MsgUpdateParams) GetSignBytes() []byte {
	return AminoCdc.MustMarshalJSON(&m)
}

func (m *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(m.Authority)
	if err != nil {
		return []sdk.AccAddress{}
	}
	return []sdk.AccAddress{addr}
}
