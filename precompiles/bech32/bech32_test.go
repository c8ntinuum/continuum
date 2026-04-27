package bech32

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestRequiredGasScalesByWords(t *testing.T) {
	precompile, err := NewPrecompile(6_000)
	require.NoError(t, err)

	require.Equal(t, uint64(6_000), precompile.RequiredGas(nil))
	require.Equal(t, uint64(6_003), precompile.RequiredGas(make([]byte, 1)))
	require.Equal(t, uint64(6_006), precompile.RequiredGas(make([]byte, 33)))
}

func TestHexToBech32RejectsLongPrefix(t *testing.T) {
	precompile, err := NewPrecompile(6_000)
	require.NoError(t, err)

	method := ABI.Methods[HexToBech32Method]
	_, err = precompile.HexToBech32(&method, []interface{}{
		common.Address{},
		strings.Repeat("a", maxBech32PrefixLength+1),
	})
	require.ErrorContains(t, err, "bech32 prefix exceeds")
}

func TestBech32ToHexRejectsLongAddress(t *testing.T) {
	precompile, err := NewPrecompile(6_000)
	require.NoError(t, err)

	method := ABI.Methods[Bech32ToHexMethod]
	_, err = precompile.Bech32ToHex(&method, []interface{}{
		strings.Repeat("a", maxBech32AddressLength+1),
	})
	require.ErrorContains(t, err, "bech32 address exceeds")
}
