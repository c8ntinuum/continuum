package json

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequiredGasScalesByWords(t *testing.T) {
	precompile, err := NewPrecompile(40_000)
	require.NoError(t, err)

	require.Equal(t, uint64(40_000), precompile.RequiredGas(nil))
	require.Equal(t, uint64(40_030), precompile.RequiredGas(make([]byte, 1)))
	require.Equal(t, uint64(40_060), precompile.RequiredGas(make([]byte, 33)))
}

func TestValidateJSONPayloadRejectsOversizeInput(t *testing.T) {
	err := validateJSONPayload(make([]byte, maxJSONInputBytes+1))
	require.ErrorContains(t, err, "json payload exceeds")
}

func TestValidateJSONPayloadRejectsExcessiveDepth(t *testing.T) {
	payload := strings.Repeat("[", maxJSONNestingDepth+1) + strings.Repeat("]", maxJSONNestingDepth+1)

	err := validateJSONPayload([]byte(payload))
	require.ErrorContains(t, err, "nesting depth")
}

func TestValidateJSONPayloadIgnoresBracketsInsideStrings(t *testing.T) {
	payload := []byte(`{"value":"[[[[[]]]]]"}`)

	require.NoError(t, validateJSONPayload(payload))
}
