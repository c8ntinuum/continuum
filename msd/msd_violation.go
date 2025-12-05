package msd

import (
	"encoding/json"
	"sync"
)

type MSDViolation struct {
	Height       int64  `json:"h"`
	Round        int32  `json:"r"`
	ProposerHex  string `json:"proposer"`
	ProposalHash string `json:"proposal"`
	Reason       string `json:"reason"`
}

type MsdViolationBuffer struct {
	mu   sync.Mutex
	data map[int64]*MSDViolation
}

func NewMsdViolationBuffer() *MsdViolationBuffer {
	return &MsdViolationBuffer{data: make(map[int64]*MSDViolation)}
}

func (b *MsdViolationBuffer) Record(h int64, r int32, proposerHex, proposalHashHex, reason string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.data[h] = &MSDViolation{
		Height: h, Round: r, ProposerHex: proposerHex, ProposalHash: proposalHashHex, Reason: reason,
	}
}

func (b *MsdViolationBuffer) ForHeight(h int64) (*MSDViolation, []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	v, exists := b.data[h]
	if !exists {
		return nil, nil
	}
	bz, _ := json.Marshal(v)
	return v, bz
}
