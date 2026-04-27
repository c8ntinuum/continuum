package utils

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestParseDelegationRejectsMalformedInputsWithoutPanic(t *testing.T) {
	validAddr := common.HexToAddress("0x1000000000000000000000000000000000000001")
	validCode := ethtypes.AddressToDelegation(validAddr)

	testCases := []struct {
		name string
		code []byte
		ok   bool
		addr common.Address
	}{
		{name: "nil", code: nil, ok: false},
		{name: "empty", code: []byte{}, ok: false},
		{name: "short prefix", code: []byte{0xef, 0x01}, ok: false},
		{name: "prefix only", code: []byte{0xef, 0x01, 0x00}, ok: false},
		{name: "too short", code: append([]byte{0xef, 0x01, 0x00}, make([]byte, 19)...), ok: false},
		{name: "too long", code: append(validCode, 0x01), ok: false},
		{name: "wrong prefix", code: append([]byte{0xef, 0x01, 0x01}, make([]byte, 20)...), ok: false},
		{name: "valid", code: validCode, ok: true, addr: validAddr},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.NotPanics(t, func() {
				addr, ok := ParseDelegation(tc.code)
				require.Equal(t, tc.ok, ok)
				require.Equal(t, tc.addr, addr)
			})
		})
	}
}
