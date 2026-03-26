package cosmos_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	protov2 "google.golang.org/protobuf/proto"

	"github.com/cosmos/evm/ante/cosmos"
	circuittype "github.com/cosmos/evm/x/circuit/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

type mockCircuitKeeper struct {
	systemAvailable bool
}

func (m mockCircuitKeeper) GetSystemAvailable(_ sdk.Context) bool {
	return m.systemAvailable
}

type mockTx struct {
	msgs []sdk.Msg
}

func (m mockTx) GetMsgs() []sdk.Msg {
	return m.msgs
}

func (m mockTx) GetMsgsV2() ([]protov2.Message, error) {
	return nil, nil
}

func (m mockTx) ValidateBasic() error {
	return nil
}

func TestCircuitAvailableDecorator(t *testing.T) {
	testCases := []struct {
		name          string
		systemEnabled bool
		tx            sdk.Tx
		expectErr     bool
	}{
		{
			name:          "system available allows EVM message",
			systemEnabled: true,
			tx:            mockTx{msgs: []sdk.Msg{&evmtypes.MsgEthereumTx{}}},
			expectErr:     false,
		},
		{
			name:          "system unavailable blocks EVM message",
			systemEnabled: false,
			tx:            mockTx{msgs: []sdk.Msg{&evmtypes.MsgEthereumTx{}}},
			expectErr:     true,
		},
		{
			name:          "system unavailable allows MsgUpdateCircuit",
			systemEnabled: false,
			tx: mockTx{msgs: []sdk.Msg{
				&circuittype.MsgUpdateCircuit{},
			}},
			expectErr: false,
		},
		{
			name:          "system unavailable blocks mixed messages",
			systemEnabled: false,
			tx: mockTx{msgs: []sdk.Msg{
				&circuittype.MsgUpdateCircuit{},
				&evmtypes.MsgEthereumTx{},
			}},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := sdk.Context{}
			nextCalled := false
			dec := cosmos.NewCircuitAvailableDecorator(mockCircuitKeeper{systemAvailable: tc.systemEnabled})

			_, err := dec.AnteHandle(ctx, tc.tx, false, func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
				nextCalled = true
				return ctx, nil
			})

			if tc.expectErr {
				require.Error(t, err)
				require.ErrorIs(t, err, sdkerrors.ErrUnauthorized)
				require.False(t, nextCalled)
				return
			}

			require.NoError(t, err)
			require.True(t, nextCalled)
		})
	}
}
