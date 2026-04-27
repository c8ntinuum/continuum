package cosmos

import (
	errorsmod "cosmossdk.io/errors"

	anteinterfaces "github.com/cosmos/evm/ante/interfaces"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/authz"
)

type IbcAvailableDecorator struct {
	ibcBreakerKeeper anteinterfaces.IbcBreakerKeeper
}

func NewIbcAvailableDecorator(ibcBreakerKeeper anteinterfaces.IbcBreakerKeeper) IbcAvailableDecorator {
	return IbcAvailableDecorator{ibcBreakerKeeper: ibcBreakerKeeper}
}

func (d IbcAvailableDecorator) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, sim bool, next sdk.AnteHandler,
) (sdk.Context, error) {
	if d.ibcBreakerKeeper.GetIbcAvailable(ctx) {
		return next(ctx, tx, sim)
	}

	for _, msg := range tx.GetMsgs() {
		isRestrictedIbcMsg, restrictedTypeURL, err := containsRestrictedIbcMsg(msg, 1)
		if err != nil {
			return ctx, err
		}
		if isRestrictedIbcMsg {
			return ctx, errorsmod.Wrapf(
				errortypes.ErrUnauthorized,
				"ibc unavailable: message type %s is restricted",
				restrictedTypeURL,
			)
		}
	}

	return next(ctx, tx, sim)
}

const maxNestedIbcMsgs = 7

var restrictedIbcMsgTypeURLs = map[string]struct{}{
	"/ibc.core.client.v1.MsgCreateClient":                                              {},
	"/ibc.core.connection.v1.MsgConnectionOpenInit":                                    {},
	"/ibc.core.channel.v1.MsgChannelOpenInit":                                          {},
	"/ibc.applications.transfer.v1.MsgTransfer":                                        {},
	"/ibc.applications.interchain_accounts.controller.v1.MsgRegisterInterchainAccount": {},
	"/ibc.applications.interchain_accounts.controller.v1.MsgSendTx":                    {},
	"/ibc.core.client.v2.MsgRegisterCounterparty":                                      {},
	"/ibc.core.client.v2.MsgUpdateClientConfig":                                        {},
	"/ibc.core.channel.v2.MsgSendPacket":                                               {},
}

func containsRestrictedIbcMsg(msg sdk.Msg, nestedLvl int) (bool, string, error) {
	if nestedLvl >= maxNestedIbcMsgs {
		return true, sdk.MsgTypeURL(msg), errorsmod.Wrapf(errortypes.ErrUnauthorized, "found more nested msgs than permitted; got: %d, expected: <%d", nestedLvl, maxNestedIbcMsgs)
	}
	switch castMsg := msg.(type) {
	case *authz.MsgExec:
		innerMsgs, err := castMsg.GetMessages()
		if err != nil {
			return true, sdk.MsgTypeURL(msg), errorsmod.Wrap(err, "failed to unpack authz messages")
		}
		for _, inner := range innerMsgs {
			if ok, typeURL, err := containsRestrictedIbcMsg(inner, nestedLvl+1); err != nil || ok {
				return ok, typeURL, err
			}
		}
		return false, "", nil
	default:
		_, isRestricted := restrictedIbcMsgTypeURLs[sdk.MsgTypeURL(msg)]
		if isRestricted {
			return true, sdk.MsgTypeURL(msg), nil
		}
		return false, "", nil
	}
}
