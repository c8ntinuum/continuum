package tokenmeta

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/cosmos/evm/precompiles/solanatx/solana/common"

	"github.com/near/borsh-go"
)

const EDITION_MARKER_BIT_SIZE uint64 = 248

type Key borsh.Enum

const (
	KeyUninitialized Key = iota
	KeyEditionV1
	KeyMasterEditionV1
	KeyReservationListV1
	KeyMetadataV1
	KeyReservationListV2
	KeyMasterEditionV2
	KeyEditionMarker
	KeyUseAuthorityRecord
	KeyCollectionAuthorityRecord
)

type Creator struct {
	Address  common.PublicKey
	Verified bool
	Share    uint8
}

type Data struct {
	Name                 string
	Symbol               string
	Uri                  string
	SellerFeeBasisPoints uint16
	Creators             *[]Creator
}

type DataV2 struct {
	Name                 string
	Symbol               string
	Uri                  string
	SellerFeeBasisPoints uint16
	Creators             *[]Creator
	Collection           *Collection
	Uses                 *Uses
}

type metadataPreV11 struct {
	Key                 Key
	UpdateAuthority     common.PublicKey
	Mint                common.PublicKey
	Data                Data
	PrimarySaleHappened bool
	IsMutable           bool
	EditionNonce        *uint8
}

type Metadata struct {
	Key                 Key
	UpdateAuthority     common.PublicKey
	Mint                common.PublicKey
	Data                Data
	PrimarySaleHappened bool
	IsMutable           bool
	EditionNonce        *uint8
	TokenStandard       *TokenStandard
	Collection          *Collection
	Uses                *Uses
	CollectionDetails   *CollectionDetails
	ProgrammableConfig  *ProgrammableConfig
}

type TokenStandard borsh.Enum

const (
	NonFungible TokenStandard = iota
	FungibleAsset
	Fungible
	NonFungibleEdition
	ProgrammableNonFungible
)

type Collection struct {
	Verified bool
	Key      common.PublicKey
}

type Uses struct {
	UseMethod UseMethod
	Remaining uint64
	Total     uint64
}

type UseMethod borsh.Enum

const (
	Burn UseMethod = iota
	Multiple
	Single
)

type CollectionDetails struct {
	Enum borsh.Enum `borsh_enum:"true"`
	V1   CollectionDetailsV1
}

type CollectionDetailsV1 struct {
	Size uint64
}

type ProgrammableConfig struct {
	Enum borsh.Enum `borsh_enum:"true"`
	V1   ProgrammableConfigV1
}

type ProgrammableConfigV1 struct {
	RuleSet *common.PublicKey
}

func MetadataDeserialize(data []byte) (Metadata, error) {
	var base metadataPreV11
	if err := borsh.Deserialize(&base, data); err != nil {
		return Metadata{}, fmt.Errorf("failed to deserialize data, err: %v", err)
	}

	metadata := Metadata{
		Key:                 base.Key,
		UpdateAuthority:     base.UpdateAuthority,
		Mint:                base.Mint,
		Data:                base.Data,
		PrimarySaleHappened: base.PrimarySaleHappened,
		IsMutable:           base.IsMutable,
		EditionNonce:        base.EditionNonce,
	}

	// borsh-go decodes Option<T> pointers as non-nil zero values when the option is None.
	// Parse optional tail fields manually to preserve nil semantics.
	if head, err := borsh.Serialize(base); err == nil && len(data) > len(head) {
		parseMetadataOptionalFields(&metadata, data[len(head):])
	}

	// trim null byte
	metadata.Data.Name = strings.TrimRight(metadata.Data.Name, "\x00")
	metadata.Data.Symbol = strings.TrimRight(metadata.Data.Symbol, "\x00")
	metadata.Data.Uri = strings.TrimRight(metadata.Data.Uri, "\x00")
	return metadata, nil
}

type metadataTailParser struct {
	data []byte
	pos  int
}

func (p *metadataTailParser) readU8() (uint8, bool) {
	if p.pos+1 > len(p.data) {
		return 0, false
	}

	v := p.data[p.pos]
	p.pos++
	return v, true
}

func (p *metadataTailParser) readBool() (bool, bool) {
	v, ok := p.readU8()
	if !ok {
		return false, false
	}

	return v == 1, true
}

func (p *metadataTailParser) readU64() (uint64, bool) {
	if p.pos+8 > len(p.data) {
		return 0, false
	}

	v := binary.LittleEndian.Uint64(p.data[p.pos : p.pos+8])
	p.pos += 8
	return v, true
}

func (p *metadataTailParser) readPublicKey() (common.PublicKey, bool) {
	if p.pos+common.PublicKeyLength > len(p.data) {
		return common.PublicKey{}, false
	}

	v := common.PublicKeyFromBytes(p.data[p.pos : p.pos+common.PublicKeyLength])
	p.pos += common.PublicKeyLength
	return v, true
}

func (p *metadataTailParser) readOptionTag() (bool, bool) {
	tag, ok := p.readU8()
	if !ok {
		return false, false
	}

	return tag == 1, true
}

func parseMetadataOptionalFields(metadata *Metadata, tail []byte) {
	p := &metadataTailParser{data: tail}

	if has, ok := p.readOptionTag(); ok && has {
		if v, ok := p.readU8(); ok {
			ts := TokenStandard(v)
			metadata.TokenStandard = &ts
		}
	}

	if has, ok := p.readOptionTag(); ok && has {
		verified, okV := p.readBool()
		key, okK := p.readPublicKey()
		if okV && okK {
			metadata.Collection = &Collection{
				Verified: verified,
				Key:      key,
			}
		}
	}

	if has, ok := p.readOptionTag(); ok && has {
		method, okM := p.readU8()
		remaining, okR := p.readU64()
		total, okT := p.readU64()
		if okM && okR && okT {
			metadata.Uses = &Uses{
				UseMethod: UseMethod(method),
				Remaining: remaining,
				Total:     total,
			}
		}
	}

	if has, ok := p.readOptionTag(); ok && has {
		enum, okE := p.readU8()
		if okE {
			cd := &CollectionDetails{Enum: borsh.Enum(enum)}
			if cd.Enum == 0 {
				if size, okS := p.readU64(); okS {
					cd.V1 = CollectionDetailsV1{Size: size}
				}
			}
			metadata.CollectionDetails = cd
		}
	}

	if has, ok := p.readOptionTag(); ok && has {
		enum, okE := p.readU8()
		if okE {
			pc := &ProgrammableConfig{Enum: borsh.Enum(enum)}
			if pc.Enum == 0 {
				if hasRuleSet, okRS := p.readOptionTag(); okRS && hasRuleSet {
					if ruleSet, okPK := p.readPublicKey(); okPK {
						pc.V1.RuleSet = &ruleSet
					}
				}
			}
			metadata.ProgrammableConfig = pc
		}
	}
}

type MasterEditionV2 struct {
	Key       Key
	Supply    uint64
	MaxSupply *uint64
}
