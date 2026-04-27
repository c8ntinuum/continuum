package common

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/require"
)

func TestCeilWords(t *testing.T) {
	testCases := []struct {
		name   string
		length uint64
		want   uint64
	}{
		{name: "zero", length: 0, want: 0},
		{name: "single byte", length: 1, want: 1},
		{name: "single word", length: 32, want: 1},
		{name: "two words", length: 33, want: 2},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, CeilWords(tc.length))
		})
	}
}

func TestLinearRequiredGasForLength(t *testing.T) {
	require.Equal(t, uint64(100), LinearRequiredGasForLength(100, 0, 30))
	require.Equal(t, uint64(130), LinearRequiredGasForLength(100, 32, 30))
	require.Equal(t, uint64(160), LinearRequiredGasForLength(100, 33, 30))
}

func TestRecoverPrecompileError(t *testing.T) {
	var err error

	func() {
		defer RecoverPrecompileError(&err)()
		panic("boom")
	}()

	require.ErrorIs(t, err, vm.ErrExecutionReverted)
}
