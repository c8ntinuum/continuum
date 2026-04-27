package types

import (
	"strings"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
)

var _ sdk.Msg = &MsgClaimRewards{}
var _ sdk.Msg = &MsgDepositRewardsPool{}
var _ sdk.Msg = &MsgUpdateParams{}
var _ sdk.Msg = &MsgSetBlocksInEpoch{}
var _ sdk.Msg = &MsgSetRewardsPerEpoch{}
var _ sdk.Msg = &MsgSetRewardingPaused{}

func (m *MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}
	if m.Params == nil {
		return errorsmod.Wrap(errortypes.ErrInvalidRequest, "params cannot be nil")
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

func (m *MsgSetBlocksInEpoch) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Signer); err != nil {
		return errorsmod.Wrap(err, "invalid signer address")
	}
	return ValidateBlocksInEpoch(m.BlocksInEpoch)
}

func (m MsgSetBlocksInEpoch) GetSignBytes() []byte {
	return AminoCdc.MustMarshalJSON(&m)
}

func (m *MsgSetBlocksInEpoch) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(m.Signer)
	if err != nil {
		return []sdk.AccAddress{}
	}
	return []sdk.AccAddress{addr}
}

func (m *MsgSetRewardsPerEpoch) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Signer); err != nil {
		return errorsmod.Wrap(err, "invalid signer address")
	}
	return ValidateRewardsPerEpoch(m.RewardsPerEpoch)
}

func (m MsgSetRewardsPerEpoch) GetSignBytes() []byte {
	return AminoCdc.MustMarshalJSON(&m)
}

func (m *MsgSetRewardsPerEpoch) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(m.Signer)
	if err != nil {
		return []sdk.AccAddress{}
	}
	return []sdk.AccAddress{addr}
}

func (m *MsgSetRewardingPaused) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Signer); err != nil {
		return errorsmod.Wrap(err, "invalid signer address")
	}
	return nil
}

func (m MsgSetRewardingPaused) GetSignBytes() []byte {
	return AminoCdc.MustMarshalJSON(&m)
}

func (m *MsgSetRewardingPaused) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(m.Signer)
	if err != nil {
		return []sdk.AccAddress{}
	}
	return []sdk.AccAddress{addr}
}

func (m *MsgClaimRewards) ValidateBasic() error {
	if err := validateAddress(m.ValidatorOperator); err != nil {
		return errorsmod.Wrap(err, "invalid validator operator address")
	}
	if err := validateAddress(m.Requester); err != nil {
		return errorsmod.Wrap(err, "invalid requester address")
	}
	return nil
}

func (m MsgClaimRewards) GetSignBytes() []byte {
	return AminoCdc.MustMarshalJSON(&m)
}

func (m *MsgClaimRewards) GetSigners() []sdk.AccAddress {
	requester, err := ParseAccAddress(m.Requester)
	if err != nil {
		return []sdk.AccAddress{}
	}
	return []sdk.AccAddress{requester}
}

func (m *MsgDepositRewardsPool) ValidateBasic() error {
	if err := validateAddress(m.Depositor); err != nil {
		return errorsmod.Wrap(err, "invalid depositor address")
	}
	if m.Amount == nil {
		return errorsmod.Wrap(errortypes.ErrInvalidCoins, "amount cannot be nil")
	}
	if !m.Amount.IsValid() {
		return errorsmod.Wrap(errortypes.ErrInvalidCoins, m.Amount.String())
	}
	if !m.Amount.IsPositive() {
		return errorsmod.Wrap(errortypes.ErrInvalidCoins, "deposit amount must be positive")
	}
	return nil
}

func (m MsgDepositRewardsPool) GetSignBytes() []byte {
	return AminoCdc.MustMarshalJSON(&m)
}

func (m *MsgDepositRewardsPool) GetSigners() []sdk.AccAddress {
	addr, err := ParseAccAddress(m.Depositor)
	if err != nil {
		return []sdk.AccAddress{}
	}
	return []sdk.AccAddress{addr}
}

func validateAddress(addr string) error {
	if strings.HasPrefix(addr, "0x") {
		if !common.IsHexAddress(addr) {
			return errortypes.ErrInvalidAddress
		}
		return nil
	}
	_, err := sdk.AccAddressFromBech32(addr)
	return err
}
func ParseAccAddress(addr string) (sdk.AccAddress, error) {
	if strings.HasPrefix(addr, "0x") {
		if !common.IsHexAddress(addr) {
			return nil, errortypes.ErrInvalidAddress
		}
		return sdk.AccAddress(common.HexToAddress(addr).Bytes()), nil
	}
	return sdk.AccAddressFromBech32(addr)
}
