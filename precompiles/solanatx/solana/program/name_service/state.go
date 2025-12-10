package name_service

import (
	"fmt"

	"github.com/cosmos/evm/precompiles/solanatx/solana/common"
)

type NameRecordHeader struct {
	ParentName common.PublicKey
	Owner      common.PublicKey
	Class      common.PublicKey
	Data       []byte
}

func NameRecordHeaderFromData(data []byte) (NameRecordHeader, error) {
	if len(data) < 96 {
		return NameRecordHeader{}, fmt.Errorf("data length should bigger than 96")
	}
	return NameRecordHeader{
		ParentName: common.PublicKeyFromBytes(data[:32]),
		Owner:      common.PublicKeyFromBytes(data[32:64]),
		Class:      common.PublicKeyFromBytes(data[64:96]),
		Data:       data[96:],
	}, nil
}
