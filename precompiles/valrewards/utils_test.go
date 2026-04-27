package valrewards

import (
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	cmn "github.com/cosmos/evm/precompiles/common"
	"github.com/stretchr/testify/require"
)

func TestParseCoinArgRejectsZeroAmount(t *testing.T) {
	_, err := parseCoinArg(cmn.Coin{
		Denom:  "ctm",
		Amount: big.NewInt(0),
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "amount must be positive")
}

func TestToCoinResponseAlwaysReturnsNonNilAmount(t *testing.T) {
	resp := toCoinResponse(sdk.Coin{})

	require.NotNil(t, resp.Amount)
	require.Zero(t, resp.Amount.Sign())
}

func TestValidatorOutstandingRewardsRejectsHexAddress(t *testing.T) {
	method := ABI.Methods[ValidatorOutstandingRewardsMethod]

	_, err := Precompile{}.ValidatorOutstandingRewards(sdk.Context{}, nil, &method, []interface{}{
		uint64(0),
		"0x0000000000000000000000000000000000000001",
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid validator address")
}
