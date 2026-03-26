package valrewards

import (
	"fmt"
	"math/big"
	"reflect"

	sdk "github.com/cosmos/cosmos-sdk/types"

	cmn "github.com/cosmos/evm/precompiles/common"
)

func parseCoinArg(v interface{}) (cmn.Coin, error) {
	if coin, ok := v.(cmn.Coin); ok {
		if coin.Denom == "" || coin.Amount == nil {
			return cmn.Coin{}, fmt.Errorf(cmn.ErrInvalidAmount, v)
		}
		return coin, nil
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return cmn.Coin{}, fmt.Errorf(cmn.ErrInvalidAmount, v)
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return cmn.Coin{}, fmt.Errorf(cmn.ErrInvalidAmount, v)
	}

	denomField := rv.FieldByName("Denom")
	amountField := rv.FieldByName("Amount")
	if !denomField.IsValid() || !amountField.IsValid() {
		return cmn.Coin{}, fmt.Errorf(cmn.ErrInvalidAmount, v)
	}

	denom, okDenom := denomField.Interface().(string)
	amount, okAmount := amountField.Interface().(*big.Int)
	if !okDenom || !okAmount || denom == "" || amount == nil {
		return cmn.Coin{}, fmt.Errorf(cmn.ErrInvalidAmount, v)
	}

	return cmn.Coin{Denom: denom, Amount: amount}, nil
}

func toCoinResponse(coin sdk.Coin) cmn.Coin {
	return cmn.Coin{
		Denom:  coin.Denom,
		Amount: coin.Amount.BigInt(),
	}
}
