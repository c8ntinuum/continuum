package utils

import (
	"encoding/binary"
	"math/big"

	evmtypes "github.com/cosmos/evm/x/vm/types"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func EncodeEpochToPay(epochToPay uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, epochToPay)
	return bz
}

func DecodeEpochToPay(bz []byte) uint64 {
	if len(bz) == 0 {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

func EncodePoints(points uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, points)
	return bz
}

func DecodePoints(bz []byte) uint64 {
	if len(bz) == 0 {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

func EncodeCoin(c sdk.Coin) []byte {
	denomBz := []byte(c.Denom)
	amountBz := c.Amount.BigInt().Bytes()
	out := make([]byte, 1+len(denomBz)+len(amountBz))
	out[0] = byte(len(denomBz))
	copy(out[1:], denomBz)
	copy(out[1+len(denomBz):], amountBz)
	return out
}

func DecodeCoin(bz []byte) sdk.Coin {
	if len(bz) == 0 {
		return sdk.Coin{
			Denom:  evmtypes.DefaultEVMDenom,
			Amount: math.ZeroInt(),
		}
	}
	denomLen := int(bz[0])
	denom := string(bz[1 : 1+denomLen])
	amountBz := bz[1+denomLen:]
	amount := decodeCoinAmount(amountBz)
	return sdk.NewCoin(denom, amount)
}

func decodeCoinAmount(bz []byte) math.Int {
	if len(bz) == 0 {
		return math.ZeroInt()
	}
	bi := new(big.Int).SetBytes(bz)
	return math.NewIntFromBigInt(bi)
}
