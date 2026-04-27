package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/evm/x/ibcratelimiterext/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ types.MsgServer = paramsMsgServer{}

type paramsMsgServer struct {
	Keeper
}

func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &paramsMsgServer{Keeper: keeper}
}

func (m paramsMsgServer) UpdateParams(goCtx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if req == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "empty request")
	}

	if m.authority.String() != req.Authority {
		return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "invalid authority")
	}

	if req.Params == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "empty params")
	}

	if err := req.Params.Validate(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	m.SetParams(ctx, *req.Params)

	return &types.MsgUpdateParamsResponse{}, nil
}
