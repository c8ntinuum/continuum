package cosmos_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/evm/ante/cosmos"
	"github.com/cosmos/evm/encoding"
	"github.com/cosmos/evm/testutil"
	"github.com/cosmos/evm/testutil/constants"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
)

type mockIbcBreakerKeeper struct {
	available bool
}

func (m mockIbcBreakerKeeper) GetIbcAvailable(_ sdk.Context) bool {
	return m.available
}

func TestIbcAvailableDecorator(t *testing.T) {
	evmConfigurator := evmtypes.NewEVMConfigurator().
		WithEVMCoinInfo(constants.ExampleChainCoinInfo[constants.ExampleChainID])
	err := evmConfigurator.Configure()
	if err != nil && !strings.Contains(err.Error(), "EVM coin info already set") {
		require.NoError(t, err)
	}

	encodingCfg := encoding.MakeConfig(constants.ExampleChainID.EVMChainID)
	txCfg := encodingCfg.TxConfig
	testPrivKeys, testAddresses, err := testutil.GeneratePrivKeyAddressPairs(2)
	require.NoError(t, err)

	evmDenom := evmtypes.GetEVMCoinDenom()
	restrictedIBCMsg := &channeltypes.MsgChannelOpenInit{}
	restrictedIBCTransferMsg := &transfertypes.MsgTransfer{}
	nonRestrictedIBCMsg := &channeltypes.MsgRecvPacket{}

	testCases := []struct {
		name        string
		msgs        []sdk.Msg
		available   bool
		expectedErr error
	}{
		{
			name:        "ibc available - allow restricted ibc msg",
			msgs:        []sdk.Msg{restrictedIBCMsg},
			available:   true,
			expectedErr: nil,
		},
		{
			name:        "ibc unavailable - reject restricted ibc msg",
			msgs:        []sdk.Msg{restrictedIBCMsg},
			available:   false,
			expectedErr: sdkerrors.ErrUnauthorized,
		},
		{
			name:        "ibc available - allow ibc transfer msg",
			msgs:        []sdk.Msg{restrictedIBCTransferMsg},
			available:   true,
			expectedErr: nil,
		},
		{
			name:        "ibc unavailable - reject ibc transfer msg",
			msgs:        []sdk.Msg{restrictedIBCTransferMsg},
			available:   false,
			expectedErr: sdkerrors.ErrUnauthorized,
		},
		{
			name:        "ibc unavailable - allow non restricted ibc msg",
			msgs:        []sdk.Msg{nonRestrictedIBCMsg},
			available:   false,
			expectedErr: nil,
		},
		{
			name: "ibc unavailable - allow non ibc msg",
			msgs: []sdk.Msg{
				banktypes.NewMsgSend(
					testAddresses[0],
					testAddresses[1],
					sdk.NewCoins(sdk.NewInt64Coin(evmDenom, 100)),
				),
			},
			available:   false,
			expectedErr: nil,
		},
		{
			name: "ibc unavailable - reject authz exec with restricted ibc msg",
			msgs: []sdk.Msg{
				testutil.NewMsgExec(testAddresses[1], []sdk.Msg{restrictedIBCMsg}),
			},
			available:   false,
			expectedErr: sdkerrors.ErrUnauthorized,
		},
		{
			name: "ibc unavailable - reject deeply nested authz exec",
			msgs: []sdk.Msg{
				testutil.CreateNestedMsgExec(testAddresses[1], 8, []sdk.Msg{restrictedIBCMsg}),
			},
			available:   false,
			expectedErr: sdkerrors.ErrUnauthorized,
		},
		{
			name: "ibc available - allow authz exec with non ibc msg",
			msgs: []sdk.Msg{
				testutil.NewMsgExec(testAddresses[1], []sdk.Msg{
					banktypes.NewMsgSend(
						testAddresses[0],
						testAddresses[1],
						sdk.NewCoins(sdk.NewInt64Coin(evmDenom, 100)),
					),
				}),
			},
			available:   true,
			expectedErr: nil,
		},
		{
			name: "ibc unavailable - allow authz grant for non ibc msg",
			msgs: []sdk.Msg{
				testutil.NewMsgGrant(
					testAddresses[0],
					testAddresses[1],
					authz.NewGenericAuthorization(sdk.MsgTypeURL(&banktypes.MsgSend{})),
					ptrTime(time.Date(9000, 1, 1, 0, 0, 0, 0, time.UTC)),
				),
			},
			available:   false,
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Case %s", tc.name), func(t *testing.T) {
			ctx := sdk.Context{}
			tx, err := testutil.CreateTx(ctx, txCfg, testPrivKeys[0], tc.msgs...)
			require.NoError(t, err)

			decorator := cosmos.NewIbcAvailableDecorator(mockIbcBreakerKeeper{available: tc.available})
			_, err = decorator.AnteHandle(ctx, tx, false, testutil.NoOpNextFn)
			if tc.expectedErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tc.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
