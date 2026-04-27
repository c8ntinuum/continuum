package utils

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestDecodeEpochToPayRejectsShortInput(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  uint64
	}{
		{name: "nil", input: nil, want: 0},
		{name: "empty", input: []byte{}, want: 0},
		{name: "short", input: []byte{0x01, 0x02, 0x03}, want: 0},
		{name: "valid", input: EncodeEpochToPay(42), want: 42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, DecodeEpochToPay(tt.input))
		})
	}
}

func TestDecodePointsRejectsShortInput(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  uint64
	}{
		{name: "nil", input: nil, want: 0},
		{name: "empty", input: []byte{}, want: 0},
		{name: "short", input: []byte{0x09, 0x08, 0x07}, want: 0},
		{name: "valid", input: EncodePoints(99), want: 99},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, DecodePoints(tt.input))
		})
	}
}

func TestEncodeCoinPanicsOnNegativeAmount(t *testing.T) {
	require.PanicsWithValue(t, "cannot encode negative coin amount", func() {
		EncodeCoin(sdk.Coin{
			Denom:  "ctm",
			Amount: math.NewInt(-1),
		})
	})
}
