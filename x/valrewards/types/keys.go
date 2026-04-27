package types

import (
	"encoding/binary"
	"fmt"
)

const (
	ModuleName = "valrewards"
	StoreKey   = "vr"
)
const (
	prefix1 = iota + 1
	prefix2
	prefix3
	prefix4
	prefix5
	prefix6
	prefix7
)

var (
	KeyPrefixEpochValidatorPoints      = []byte{prefix1}
	KeyPrefixEpochValidatorOutstanding = []byte{prefix2}
	KeyPrefixEpochToPay                = []byte{prefix3}
	KeyParams                          = []byte{prefix4}
	KeyCurrentRewardSettings           = []byte{prefix5}
	KeyNextRewardSettings              = []byte{prefix6}
	KeyEpochState                      = []byte{prefix7}
)

func GetEpochValidatorPointsKey(epoch uint64, addressBytes []byte) []byte {
	var epochBz [8]byte
	binary.BigEndian.PutUint64(epochBz[:], epoch)

	bz := append([]byte{}, KeyPrefixEpochValidatorPoints...)
	bz = append(bz, epochBz[:]...)
	return append(bz, addressBytes...)
}

func GetEpochValidatorPointsListKey(epoch uint64) []byte {
	var epochBz [8]byte
	binary.BigEndian.PutUint64(epochBz[:], epoch)

	bz := append([]byte{}, KeyPrefixEpochValidatorPoints...)
	return append(bz, epochBz[:]...)
}

func GetEpochValidatorOutstandingKey(epoch uint64, addressBytes []byte) []byte {
	var epochBz [8]byte
	binary.BigEndian.PutUint64(epochBz[:], epoch)

	bz := append([]byte{}, KeyPrefixEpochValidatorOutstanding...)
	bz = append(bz, epochBz[:]...)
	return append(bz, addressBytes...)
}

func GetEpochValidatorOutstandingListKey(epoch uint64) []byte {
	var epochBz [8]byte
	binary.BigEndian.PutUint64(epochBz[:], epoch)

	bz := append([]byte{}, KeyPrefixEpochValidatorOutstanding...)
	return append(bz, epochBz[:]...)
}

func GetEpochToPayKey() []byte {
	return KeyPrefixEpochToPay
}

func GetParamsKey() []byte {
	return KeyParams
}

func GetCurrentRewardSettingsKey() []byte {
	return KeyCurrentRewardSettings
}

func GetNextRewardSettingsKey() []byte {
	return KeyNextRewardSettings
}

func GetEpochStateKey() []byte {
	return KeyEpochState
}

func ParseEpochValidatorPointsKey(key []byte) (uint64, string, error) {
	return parseEpochValidatorKey(KeyPrefixEpochValidatorPoints, key)
}

func ParseEpochValidatorOutstandingKey(key []byte) (uint64, string, error) {
	return parseEpochValidatorKey(KeyPrefixEpochValidatorOutstanding, key)
}

func parseEpochValidatorKey(prefix, key []byte) (uint64, string, error) {
	if len(key) < len(prefix)+8+1 {
		return 0, "", fmt.Errorf("invalid key length %d", len(key))
	}
	if string(key[:len(prefix)]) != string(prefix) {
		return 0, "", fmt.Errorf("invalid key prefix")
	}

	epoch := binary.BigEndian.Uint64(key[len(prefix) : len(prefix)+8])
	validatorAddress := string(key[len(prefix)+8:])
	if validatorAddress == "" {
		return 0, "", fmt.Errorf("missing validator address")
	}

	return epoch, validatorAddress, nil
}
