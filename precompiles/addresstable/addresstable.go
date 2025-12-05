package addresstable

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"errors"
	"math/big"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cmn "github.com/cosmos/evm/precompiles/common"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
)

// Compile-time check.
var _ vm.PrecompiledContract = &Precompile{}

var (
	// Embed abi json file to the executable binary. Needed when importing as dependency.
	//
	//go:embed abi.json
	f   []byte
	ABI abi.ABI
)

func init() {
	var err error
	ABI, err = abi.JSON(bytes.NewReader(f))
	if err != nil {
		panic(err)
	}
}

const (
	methodAddressExists = "addressExists"
	methodCompress      = "compress"
	methodDecompress    = "decompress"
	methodLookup        = "lookup"
	methodLookupIndex   = "lookupIndex"
	methodRegister      = "register"
	methodSize          = "size"
)

// Storage schema (in this contract's storage space):
//
//	slotSize            := keccak256("addr.size")
//	slotAddrToIndex(a)  := keccak256(0x01 || a[20])
//	slotIndexToAddr(i)  := keccak256(0x02 || uint256(i))
//
// In addr->index we store (index + 1) so "0" unambiguously means "absent".
var (
	slotSizeKey = keccakString("addr.size")
)

type Precompile struct {
	abi.ABI
	cmn.Precompile
}

func NewPrecompile(bankKeeper cmn.BankKeeper) (*Precompile, error) {
	return &Precompile{ABI: ABI, Precompile: cmn.Precompile{
		KvGasConfig:           storetypes.KVGasConfig(),
		TransientKVGasConfig:  storetypes.TransientGasConfig(),
		ContractAddress:       common.HexToAddress(evmtypes.AddressTablePrecompileAddress),
		BalanceHandlerFactory: cmn.NewBalanceHandlerFactory(bankKeeper),
	}}, nil
}

func (Precompile) Address() gethcommon.Address {
	return gethcommon.HexToAddress(evmtypes.AddressTablePrecompileAddress)

}

func (p Precompile) RequiredGas(input []byte) uint64 {
	if len(input) < 4 {
		return 0
	}
	methodID := input[:4]
	method, err := p.MethodById(methodID)
	if err != nil {
		return 0
	}
	return p.Precompile.RequiredGas(input, p.IsTransaction(method))
}

func (p Precompile) Run(evm *vm.EVM, contract *vm.Contract, readonly bool) ([]byte, error) {
	return p.RunNativeAction(evm, contract, func(ctx sdk.Context) ([]byte, error) {
		return p.Execute(evm, contract, readonly)
	})
}

func (p *Precompile) Execute(evm *vm.EVM, contract *vm.Contract, _ bool) ([]byte, error) {
	input := contract.Input
	if len(input) < 4 {
		return nil, vm.ErrExecutionReverted
	}
	m, err := p.MethodById(input[:4])
	if err != nil {
		return nil, vm.ErrExecutionReverted
	}
	switch m.Name {
	case methodAddressExists:
		return p.addressExists(evm, contract, m, input[4:])
	case methodCompress:
		return p.compress(evm, contract, m, input[4:])
	case methodDecompress:
		return p.decompress(evm, contract, m, input[4:])
	case methodLookup:
		return p.lookup(evm, contract, m, input[4:])
	case methodLookupIndex:
		return p.lookupIndex(evm, contract, m, input[4:])
	case methodRegister:
		return p.register(evm, contract, m, input[4:])
	case methodSize:
		return p.size(evm, contract, m, input[4:])
	default:
		return nil, vm.ErrExecutionReverted
	}
}

func getSize(evm *vm.EVM, self gethcommon.Address) uint64 {
	h := evm.StateDB.GetState(self, slotSizeKey)
	if h == (gethcommon.Hash{}) {
		return 0
	}
	return new(big.Int).SetBytes(h.Bytes()).Uint64()
}

func putSize(evm *vm.EVM, self gethcommon.Address, n uint64) {
	var b [32]byte
	bi := new(big.Int).SetUint64(n)
	copy(b[32-len(bi.Bytes()):], bi.Bytes())
	evm.StateDB.SetState(self, slotSizeKey, gethcommon.BytesToHash(b[:]))
}

func addrIndexSlot(a gethcommon.Address) gethcommon.Hash {
	key := append([]byte{0x01}, a.Bytes()...)
	return keccak(key)
}

func indexAddrSlot(i uint64) gethcommon.Hash {
	var num [32]byte
	binary.BigEndian.PutUint64(num[24:], i)
	key := append([]byte{0x02}, num[:]...)
	return keccak(key)
}

func getIndexOf(evm *vm.EVM, self gethcommon.Address, a gethcommon.Address) (uint64, bool) {
	slot := addrIndexSlot(a)
	v := evm.StateDB.GetState(self, slot)
	if v == (gethcommon.Hash{}) {
		return 0, false
	}
	idxPlus1 := new(big.Int).SetBytes(v.Bytes()).Uint64()
	if idxPlus1 == 0 {
		return 0, false
	}
	return idxPlus1 - 1, true
}

func setIndexOf(evm *vm.EVM, self gethcommon.Address, a gethcommon.Address, index uint64) {
	var b [32]byte
	bi := new(big.Int).SetUint64(index + 1) // store index+1
	copy(b[32-len(bi.Bytes()):], bi.Bytes())
	evm.StateDB.SetState(self, addrIndexSlot(a), gethcommon.BytesToHash(b[:]))
}

func getAddressAt(evm *vm.EVM, self gethcommon.Address, index uint64) (gethcommon.Address, bool) {
	slot := indexAddrSlot(index)
	h := evm.StateDB.GetState(self, slot)
	if h == (gethcommon.Hash{}) {
		return gethcommon.Address{}, false
	}
	var a gethcommon.Address
	copy(a[:], h.Bytes()[12:])
	return a, true
}

func setAddressAt(evm *vm.EVM, self gethcommon.Address, index uint64, a gethcommon.Address) {
	var b [32]byte
	copy(b[12:], a.Bytes())
	evm.StateDB.SetState(self, indexAddrSlot(index), gethcommon.BytesToHash(b[:]))
}

func (p *Precompile) addressExists(evm *vm.EVM, contract *vm.Contract, m *abi.Method, data []byte) ([]byte, error) {
	vals, err := m.Inputs.Unpack(data)
	if err != nil || len(vals) != 1 {
		return nil, vm.ErrExecutionReverted
	}
	addr, ok := vals[0].(gethcommon.Address)
	if !ok {
		out, _ := m.Outputs.Pack(false)
		return out, nil
	}
	_, exists := getIndexOf(evm, contract.Address(), addr)
	out, _ := m.Outputs.Pack(exists)
	return out, nil
}

func (p *Precompile) size(evm *vm.EVM, contract *vm.Contract, m *abi.Method, data []byte) ([]byte, error) {
	if len(data) != 0 {
		return nil, vm.ErrExecutionReverted
	}
	out, _ := m.Outputs.Pack(new(big.Int).SetUint64(getSize(evm, contract.Address())))
	return out, nil
}

func (p *Precompile) lookup(evm *vm.EVM, contract *vm.Contract, m *abi.Method, data []byte) ([]byte, error) {
	vals, err := m.Inputs.Unpack(data)
	if err != nil || len(vals) != 1 {
		return nil, vm.ErrExecutionReverted
	}
	addr, ok := vals[0].(gethcommon.Address)
	if !ok {
		return nil, vm.ErrExecutionReverted
	}
	index, exists := getIndexOf(evm, contract.Address(), addr)
	if !exists {
		return nil, errors.New("address does not exist in AddressTable")
	}
	out, _ := m.Outputs.Pack(new(big.Int).SetUint64(index))
	return out, nil
}

func (p *Precompile) lookupIndex(evm *vm.EVM, contract *vm.Contract, m *abi.Method, data []byte) ([]byte, error) {
	vals, err := m.Inputs.Unpack(data)
	if err != nil || len(vals) != 1 {
		return nil, vm.ErrExecutionReverted
	}
	indexBI, ok := vals[0].(*big.Int)
	if !ok || indexBI.Sign() < 0 {
		return nil, vm.ErrExecutionReverted
	}
	index := indexBI.Uint64()
	a, exists := getAddressAt(evm, contract.Address(), index)
	if !exists {
		return nil, errors.New("index does not exist in AddressTable")
	}
	out, _ := m.Outputs.Pack(a)
	return out, nil
}

func (p *Precompile) register(evm *vm.EVM, contract *vm.Contract, m *abi.Method, data []byte) ([]byte, error) {
	vals, err := m.Inputs.Unpack(data)
	if err != nil || len(vals) != 1 {
		return nil, vm.ErrExecutionReverted
	}
	addr, ok := vals[0].(gethcommon.Address)
	if !ok {
		return nil, vm.ErrExecutionReverted
	}
	if idx, exists := getIndexOf(evm, contract.Address(), addr); exists {
		out, _ := m.Outputs.Pack(new(big.Int).SetUint64(idx))
		return out, nil
	}

	size := getSize(evm, contract.Address())
	setIndexOf(evm, contract.Address(), addr, size)
	setAddressAt(evm, contract.Address(), size, addr)
	putSize(evm, contract.Address(), size+1)

	out, _ := m.Outputs.Pack(new(big.Int).SetUint64(size))
	return out, nil
}

func (p *Precompile) compress(evm *vm.EVM, contract *vm.Contract, m *abi.Method, data []byte) ([]byte, error) {
	vals, err := m.Inputs.Unpack(data)
	if err != nil || len(vals) != 1 {
		return nil, vm.ErrExecutionReverted
	}
	addr, ok := vals[0].(gethcommon.Address)
	if !ok {
		return nil, vm.ErrExecutionReverted
	}

	if idx, exists := getIndexOf(evm, contract.Address(), addr); exists {
		payload := encodeCompressedIndex(idx)
		out, _ := m.Outputs.Pack(payload)
		return out, nil
	}

	payload := encodeCompressedUnregistered(addr)
	out, _ := m.Outputs.Pack(payload)
	return out, nil
}

func (p *Precompile) decompress(evm *vm.EVM, contract *vm.Contract, m *abi.Method, data []byte) ([]byte, error) {
	vals, err := m.Inputs.Unpack(data)
	if err != nil || len(vals) != 2 {
		return nil, vm.ErrExecutionReverted
	}
	buf, ok1 := vals[0].([]byte)
	offsetBI, ok2 := vals[1].(*big.Int)
	if !(ok1 && ok2) {
		return nil, vm.ErrExecutionReverted
	}
	if offsetBI.Sign() < 0 {
		return nil, errors.New("invalid offset")
	}
	offset := offsetBI.Uint64()

	addr, newOff, derr := decodeCompressed(buf, offset, func(idx uint64) (gethcommon.Address, bool) {
		return getAddressAt(evm, contract.Address(), idx)
	})
	if derr != nil {
		return nil, derr
	}
	out, _ := m.Outputs.Pack(addr, new(big.Int).SetUint64(newOff))
	return out, nil
}

func encodeCompressedIndex(index uint64) []byte {
	var be [8]byte
	binary.BigEndian.PutUint64(be[:], index)
	i := 0
	for i < 7 && be[i] == 0x00 {
		i++
	}
	min := be[i:]
	out := make([]byte, 0, 1+1+len(min))
	out = append(out, 0x00)
	out = append(out, byte(len(min)))
	out = append(out, min...)
	return out
}

func encodeCompressedUnregistered(a gethcommon.Address) []byte {
	out := make([]byte, 0, 1+20)
	out = append(out, 0xff)
	out = append(out, a.Bytes()...)
	return out
}

func decodeCompressed(buf []byte, offset uint64, indexToAddr func(uint64) (gethcommon.Address, bool)) (gethcommon.Address, uint64, error) {
	if offset >= uint64(len(buf)) {
		return gethcommon.Address{}, 0, errors.New("invalid offset in decompress")
	}
	tag := buf[offset]
	switch tag {
	case 0x00:
		if offset+1 >= uint64(len(buf)) {
			return gethcommon.Address{}, 0, errors.New("invalid offset in decompress")
		}
		l := uint64(buf[offset+1])
		start := offset + 2
		end := start + l
		if end > uint64(len(buf)) || l == 0 || l > 8 {
			return gethcommon.Address{}, 0, errors.New("invalid index encoding")
		}
		var be [8]byte
		copy(be[8-l:], buf[start:end])
		idx := binary.BigEndian.Uint64(be[:])
		a, ok := indexToAddr(idx)
		if !ok {
			return gethcommon.Address{}, 0, errors.New("index does not exist in AddressTable")
		}
		return a, end, nil
	case 0xff:
		start := offset + 1
		end := start + 20
		if end > uint64(len(buf)) {
			return gethcommon.Address{}, 0, errors.New("invalid raw address encoding")
		}
		var a gethcommon.Address
		copy(a[:], buf[start:end])
		return a, end, nil
	default:
		return gethcommon.Address{}, 0, errors.New("unknown compression tag")
	}
}

func keccak(b []byte) gethcommon.Hash {
	return crypto.Keccak256Hash(b)
}

func keccakString(s string) gethcommon.Hash {
	return keccak([]byte(s))
}

func (Precompile) IsTransaction(method *abi.Method) bool {
	switch method.Name {
	case methodRegister:
		return true
	default:
		return false
	}
}
