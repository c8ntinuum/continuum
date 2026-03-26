package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	ratelimittypes "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/types"
)

var _ ratelimittypes.MsgServer = msgServer{}

type rateLimitKeeper interface {
	GetAuthority() string
	AddRateLimit(ctx sdk.Context, msg *ratelimittypes.MsgAddRateLimit) error
	UpdateRateLimit(ctx sdk.Context, msg *ratelimittypes.MsgUpdateRateLimit) error
	RemoveRateLimit(ctx sdk.Context, denom, channelID string)
	GetRateLimit(ctx sdk.Context, denom, channelID string) (ratelimittypes.RateLimit, bool)
	ResetRateLimit(ctx sdk.Context, denom, channelID string) error
}

type whitelistKeeper interface {
	IsWhitelisted(ctx sdk.Context, addr sdk.AccAddress) bool
}

type msgServer struct {
	rateLimitKeeper rateLimitKeeper
	whitelistKeeper whitelistKeeper
}

func NewRateLimitMsgServer(rateLimitKeeper rateLimitKeeper, whitelistKeeper whitelistKeeper) ratelimittypes.MsgServer {
	return &msgServer{
		rateLimitKeeper: rateLimitKeeper,
		whitelistKeeper: whitelistKeeper,
	}
}

func (m msgServer) isAuthorized(ctx sdk.Context, signer string) error {
	addr, err := sdk.AccAddressFromBech32(signer)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid signer address: %s", signer)
	}

	if m.rateLimitKeeper.GetAuthority() == signer {
		return nil
	}

	if m.whitelistKeeper.IsWhitelisted(ctx, addr) {
		return nil
	}

	return errorsmod.Wrapf(govtypes.ErrInvalidSigner, "unauthorized signer; expected authority or whitelisted address, got %s", signer)
}

func (m msgServer) AddRateLimit(goCtx context.Context, msg *ratelimittypes.MsgAddRateLimit) (*ratelimittypes.MsgAddRateLimitResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := m.isAuthorized(ctx, msg.Authority); err != nil {
		return nil, err
	}

	if err := m.rateLimitKeeper.AddRateLimit(ctx, msg); err != nil {
		return nil, err
	}

	return &ratelimittypes.MsgAddRateLimitResponse{}, nil
}

func (m msgServer) UpdateRateLimit(goCtx context.Context, msg *ratelimittypes.MsgUpdateRateLimit) (*ratelimittypes.MsgUpdateRateLimitResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := m.isAuthorized(ctx, msg.Authority); err != nil {
		return nil, err
	}

	if err := m.rateLimitKeeper.UpdateRateLimit(ctx, msg); err != nil {
		return nil, err
	}

	return &ratelimittypes.MsgUpdateRateLimitResponse{}, nil
}

func (m msgServer) RemoveRateLimit(goCtx context.Context, msg *ratelimittypes.MsgRemoveRateLimit) (*ratelimittypes.MsgRemoveRateLimitResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := m.isAuthorized(ctx, msg.Authority); err != nil {
		return nil, err
	}

	_, found := m.rateLimitKeeper.GetRateLimit(ctx, msg.Denom, msg.ChannelOrClientId)
	if !found {
		return nil, ratelimittypes.ErrRateLimitNotFound
	}

	m.rateLimitKeeper.RemoveRateLimit(ctx, msg.Denom, msg.ChannelOrClientId)
	return &ratelimittypes.MsgRemoveRateLimitResponse{}, nil
}

func (m msgServer) ResetRateLimit(goCtx context.Context, msg *ratelimittypes.MsgResetRateLimit) (*ratelimittypes.MsgResetRateLimitResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := m.isAuthorized(ctx, msg.Authority); err != nil {
		return nil, err
	}

	if err := m.rateLimitKeeper.ResetRateLimit(ctx, msg.Denom, msg.ChannelOrClientId); err != nil {
		return nil, err
	}

	return &ratelimittypes.MsgResetRateLimitResponse{}, nil
}
