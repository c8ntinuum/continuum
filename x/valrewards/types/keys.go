package types

import (
	"encoding/binary"
)

const (
	ModuleName = "valrewards"
	StoreKey   = "vr"
)

// prefix bytes for the feedistribution persistent store
const (
	prefix1 = iota + 1
	prefix2
	prefix3
)

// KVStore key prefixes
var (
	KeyPrefixEpochValidatorPoints      = []byte{prefix1}
	KeyPrefixEpochValidatorOutstanding = []byte{prefix2}
	KeyPrefixEpochToPay                = []byte{prefix3}
)

func GetEpochValidatorPointsKey(epoch uint64, addresBytes []byte) []byte {
	var epochBz [8]byte
	binary.BigEndian.PutUint64(epochBz[:], epoch)

	bz := KeyPrefixEpochValidatorPoints
	bz = append(bz, epochBz[:]...)
	return append(bz, addresBytes...)
}

func GetEpochValidatorPointsListKey(epoch uint64) []byte {
	var epochBz [8]byte
	binary.BigEndian.PutUint64(epochBz[:], epoch)

	bz := KeyPrefixEpochValidatorPoints
	return append(bz, epochBz[:]...)
}

func GetEpochValidatorOutstandingKey(epoch uint64, addresBytes []byte) []byte {
	var epochBz [8]byte
	binary.BigEndian.PutUint64(epochBz[:], epoch)

	bz := KeyPrefixEpochValidatorOutstanding
	bz = append(bz, epochBz[:]...)
	return append(bz, addresBytes...)
}

func GetEpochValidatorOutstandingListKey(epoch uint64) []byte {
	var epochBz [8]byte
	binary.BigEndian.PutUint64(epochBz[:], epoch)

	bz := KeyPrefixEpochValidatorOutstanding
	return append(bz, epochBz[:]...)
}

func GetEpochToPayKey() []byte {
	return KeyPrefixEpochToPay
}
