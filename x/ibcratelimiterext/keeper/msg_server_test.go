package keeper

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	ratelimittypes "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/types"
)

type fakeRateLimitKeeper struct {
	authority    string
	addCalled    bool
	updateCalled bool
	removeCalled bool
	resetCalled  bool
	found        bool
	addErr       error
	updateErr    error
	resetErr     error
}

func (f *fakeRateLimitKeeper) GetAuthority() string { return f.authority }
func (f *fakeRateLimitKeeper) AddRateLimit(_ sdk.Context, _ *ratelimittypes.MsgAddRateLimit) error {
	f.addCalled = true
	return f.addErr
}
func (f *fakeRateLimitKeeper) UpdateRateLimit(_ sdk.Context, _ *ratelimittypes.MsgUpdateRateLimit) error {
	f.updateCalled = true
	return f.updateErr
}
func (f *fakeRateLimitKeeper) RemoveRateLimit(_ sdk.Context, _, _ string) {
	f.removeCalled = true
}
func (f *fakeRateLimitKeeper) GetRateLimit(_ sdk.Context, _, _ string) (ratelimittypes.RateLimit, bool) {
	return ratelimittypes.RateLimit{
		Path: &ratelimittypes.Path{
			Denom:             "ibc/test",
			ChannelOrClientId: "channel-0",
		},
		Quota: &ratelimittypes.Quota{
			MaxPercentSend: sdkmath.NewInt(10),
			MaxPercentRecv: sdkmath.NewInt(10),
			DurationHours:  24,
		},
		Flow: &ratelimittypes.Flow{
			Inflow:       sdkmath.ZeroInt(),
			Outflow:      sdkmath.ZeroInt(),
			ChannelValue: sdkmath.NewInt(100),
		},
	}, f.found
}
func (f *fakeRateLimitKeeper) ResetRateLimit(_ sdk.Context, _, _ string) error {
	f.resetCalled = true
	return f.resetErr
}

type fakeWhitelistKeeper struct {
	whitelisted map[string]bool
}

func (f *fakeWhitelistKeeper) IsWhitelisted(_ sdk.Context, addr sdk.AccAddress) bool {
	return f.whitelisted[addr.String()]
}

func testContext() sdk.Context {
	return sdk.Context{}.WithContext(context.Background())
}

func setBech32Prefix() {
	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount("cosmos", "cosmospub")
}

func TestAddRateLimit_AuthorityAllowed(t *testing.T) {
	setBech32Prefix()
	authority := sdk.AccAddress([]byte("authority_address_123")).String()
	rlk := &fakeRateLimitKeeper{authority: authority}
	wlk := &fakeWhitelistKeeper{whitelisted: map[string]bool{}}
	srv := NewRateLimitMsgServer(rlk, wlk)

	resp, err := srv.AddRateLimit(
		testContext(),
		&ratelimittypes.MsgAddRateLimit{
			Authority:         authority,
			Denom:             "ibc/test",
			ChannelOrClientId: "channel-0",
			MaxPercentSend:    sdkmath.NewInt(10),
			MaxPercentRecv:    sdkmath.NewInt(10),
			DurationHours:     24,
		},
	)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, rlk.addCalled)
}

func TestAddRateLimit_AuthorityAllowedWithCanonicalizedCase(t *testing.T) {
	setBech32Prefix()
	authority := sdk.AccAddress([]byte("authority_address_123")).String()
	rlk := &fakeRateLimitKeeper{authority: authority}
	wlk := &fakeWhitelistKeeper{whitelisted: map[string]bool{}}
	srv := NewRateLimitMsgServer(rlk, wlk)

	resp, err := srv.AddRateLimit(
		testContext(),
		&ratelimittypes.MsgAddRateLimit{
			Authority:         strings.ToUpper(authority),
			Denom:             "ibc/test",
			ChannelOrClientId: "channel-0",
			MaxPercentSend:    sdkmath.NewInt(10),
			MaxPercentRecv:    sdkmath.NewInt(10),
			DurationHours:     24,
		},
	)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, rlk.addCalled)
}

func TestAddRateLimit_WhitelistedAllowed(t *testing.T) {
	setBech32Prefix()
	authority := sdk.AccAddress([]byte("authority_address_123")).String()
	operator := sdk.AccAddress([]byte("whitelisted_operator")).String()
	rlk := &fakeRateLimitKeeper{authority: authority}
	wlk := &fakeWhitelistKeeper{whitelisted: map[string]bool{operator: true}}
	srv := NewRateLimitMsgServer(rlk, wlk)

	resp, err := srv.AddRateLimit(
		testContext(),
		&ratelimittypes.MsgAddRateLimit{
			Authority:         operator,
			Denom:             "ibc/test",
			ChannelOrClientId: "channel-0",
			MaxPercentSend:    sdkmath.NewInt(10),
			MaxPercentRecv:    sdkmath.NewInt(10),
			DurationHours:     24,
		},
	)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, rlk.addCalled)
}

func TestAddRateLimit_UnauthorizedDenied(t *testing.T) {
	setBech32Prefix()
	authority := sdk.AccAddress([]byte("authority_address_123")).String()
	unauthorized := sdk.AccAddress([]byte("unauthorized_operator")).String()
	rlk := &fakeRateLimitKeeper{authority: authority}
	wlk := &fakeWhitelistKeeper{whitelisted: map[string]bool{}}
	srv := NewRateLimitMsgServer(rlk, wlk)

	resp, err := srv.AddRateLimit(
		testContext(),
		&ratelimittypes.MsgAddRateLimit{
			Authority:         unauthorized,
			Denom:             "ibc/test",
			ChannelOrClientId: "channel-0",
			MaxPercentSend:    sdkmath.NewInt(10),
			MaxPercentRecv:    sdkmath.NewInt(10),
			DurationHours:     24,
		},
	)

	require.Error(t, err)
	require.Nil(t, resp)
	require.False(t, rlk.addCalled)
}

func TestRemoveRateLimit_NotFound(t *testing.T) {
	setBech32Prefix()
	authority := sdk.AccAddress([]byte("authority_address_123")).String()
	rlk := &fakeRateLimitKeeper{authority: authority, found: false}
	wlk := &fakeWhitelistKeeper{whitelisted: map[string]bool{}}
	srv := NewRateLimitMsgServer(rlk, wlk)

	resp, err := srv.RemoveRateLimit(
		testContext(),
		&ratelimittypes.MsgRemoveRateLimit{
			Authority:         authority,
			Denom:             "ibc/test",
			ChannelOrClientId: "channel-0",
		},
	)

	require.ErrorIs(t, err, ratelimittypes.ErrRateLimitNotFound)
	require.Nil(t, resp)
	require.False(t, rlk.removeCalled)
}

func TestResetRateLimit_WhitelistedAllowed(t *testing.T) {
	setBech32Prefix()
	authority := sdk.AccAddress([]byte("authority_address_123")).String()
	operator := sdk.AccAddress([]byte("whitelisted_operator")).String()
	rlk := &fakeRateLimitKeeper{authority: authority}
	wlk := &fakeWhitelistKeeper{whitelisted: map[string]bool{operator: true}}
	srv := NewRateLimitMsgServer(rlk, wlk)

	resp, err := srv.ResetRateLimit(
		testContext(),
		&ratelimittypes.MsgResetRateLimit{
			Authority:         operator,
			Denom:             "ibc/test",
			ChannelOrClientId: "channel-0",
		},
	)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, rlk.resetCalled)
}
