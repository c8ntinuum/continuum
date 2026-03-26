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

func (m *MsgClaimRewards) ValidateBasic() error {
	if err := validateAddress(m.Delegator); err != nil {
		return errorsmod.Wrap(err, "invalid delegator address")
	}
	return nil
}

func (m MsgClaimRewards) GetSignBytes() []byte {
	return AminoCdc.MustMarshalJSON(&m)
}

func (m *MsgClaimRewards) GetSigners() []sdk.AccAddress {
	addr, err := ParseAccAddress(m.Delegator)
	if err != nil {
		return []sdk.AccAddress{}
	}
	return []sdk.AccAddress{addr}
}

func (m *MsgDepositRewardsPool) ValidateBasic() error {
	if err := validateAddress(m.Depositor); err != nil {
		return errorsmod.Wrap(err, "invalid depositor address")
	}
	if !m.Amount.IsValid() {
		return errorsmod.Wrap(errortypes.ErrInvalidCoins, m.Amount.String())
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
