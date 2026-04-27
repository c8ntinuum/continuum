package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKeyBuildersDoNotMutateSharedPrefixes(t *testing.T) {
	originalPointsPrefix := append([]byte{}, KeyPrefixEpochValidatorPoints...)
	originalOutstandingPrefix := append([]byte{}, KeyPrefixEpochValidatorOutstanding...)

	_ = GetEpochValidatorPointsKey(7, []byte("validator-a"))
	_ = GetEpochValidatorPointsListKey(7)
	_ = GetEpochValidatorOutstandingKey(9, []byte("validator-b"))
	_ = GetEpochValidatorOutstandingListKey(9)

	require.Equal(t, originalPointsPrefix, KeyPrefixEpochValidatorPoints)
	require.Equal(t, originalOutstandingPrefix, KeyPrefixEpochValidatorOutstanding)
}
