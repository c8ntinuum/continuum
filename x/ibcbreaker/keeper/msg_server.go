package keeper

import (
	"context"

	"github.com/cosmos/evm/x/ibcbreaker/types"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ types.MsgServer = msgServer{}

type msgServer struct {
	Keeper
}

func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

func (m msgServer) UpdateIbcBreaker(goCtx context.Context, req *types.MsgUpdateIbcBreaker) (*types.MsgUpdateIbcBreakerResponse, error) {
	if req == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	signer, err := sdk.AccAddressFromBech32(req.Signer)
	if err != nil {
		return nil, errorsmod.Wrap(err, "invalid signer address")
	}

	if !m.IsWhitelisted(ctx, signer) {
		return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "signer not whitelisted")
	}

	if m.GetIbcAvailable(ctx) == req.IbcAvailable {
		return &types.MsgUpdateIbcBreakerResponse{}, nil
	}

	m.SetIbcAvailable(ctx, req.IbcAvailable)

	return &types.MsgUpdateIbcBreakerResponse{}, nil
}

func (m msgServer) UpdateParams(goCtx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
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
