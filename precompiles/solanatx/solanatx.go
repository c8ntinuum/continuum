package solanatx

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"fmt"
	"math/big"
	"reflect"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cmn "github.com/cosmos/evm/precompiles/common"
	cmnsol "github.com/cosmos/evm/precompiles/solanatx/solana/common"
	"github.com/cosmos/evm/precompiles/solanatx/solana/program/compute_budget"
	stypes "github.com/cosmos/evm/precompiles/solanatx/solana/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	generateSolanaTx = "generateSolanaTx"
	getSolanaTx      = "getSolanaTx"
	getSolanaTxs     = "getSolanaTxs"
)

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

func (Precompile) Address() common.Address {
	return common.HexToAddress(evmtypes.SolanaTxPrecompileAddress)
}

func (p Precompile) RequiredGas(input []byte) uint64 {
	if len(input) < 4 {
		return 0
	}
	methodID := input[:4]
	method, err := p.MethodById(methodID)
	if err != nil {
		// This should never happen since this method is going to fail during Run
		return 0
	}
	return p.Precompile.RequiredGas(input, p.IsTransaction(method))
}
func (p Precompile) Run(evm *vm.EVM, contract *vm.Contract, readonly bool) ([]byte, error) {
	return p.RunNativeAction(evm, contract, func(ctx sdk.Context) ([]byte, error) {
		return p.Execute(evm, contract, readonly)
	})
}

func (p *Precompile) Execute(evm *vm.EVM, contract *vm.Contract, _ bool) (res []byte, err error) {
	if len(contract.Input) < 4 {
		return nil, vm.ErrExecutionReverted
	}
	methodID := contract.Input[:4]
	method, err := p.MethodById(methodID)
	if err != nil {
		return nil, err
	}
	argsBz := contract.Input[4:]
	args, err := method.Inputs.Unpack(argsBz)
	if err != nil {
		return nil, err
	}

	switch method.Name {
	case generateSolanaTx:
		res, err = p.generateSolanaTx(evm, contract, method, args)
	case getSolanaTx:
		res, err = p.getSolanaTx(evm, contract, method, args)
	case getSolanaTxs:
		res, err = p.getSolanaTxs(evm, contract, method, args)
	}
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (p Precompile) generateSolanaTx(
	evm *vm.EVM,
	contract *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	// 0: string priorityLevel
	// 1: bytes32 programKey
	// 2: bytes32 signer
	// 3: bytes methodParams
	// 4: tuple[] accounts (pubKey bytes32, isSigner bool, isWritable bool)
	// 5: string latestBlockHash
	if len(args) != 6 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 6, len(args))
	}

	priorityLevel := args[0].(string)
	programKey := args[1].([32]byte)
	signer := args[2].([32]byte)
	methodParams := args[3].([]byte)

	// decode accounts
	rawAccounts := reflect.ValueOf(args[4])
	if rawAccounts.Kind() != reflect.Slice {
		return nil, fmt.Errorf("accounts is not a slice")
	}
	accounts := make([]AccountMetaAbi, rawAccounts.Len())
	for i := 0; i < rawAccounts.Len(); i++ {
		elem := rawAccounts.Index(i)
		accounts[i] = AccountMetaAbi{
			PubKey:     elem.FieldByName("PubKey").Interface().([32]byte),
			IsSigner:   elem.FieldByName("IsSigner").Interface().(bool),
			IsWritable: elem.FieldByName("IsWritable").Interface().(bool),
		}
	}

	latestBlockHash := args[5].(string)

	// build compute budget by priority
	var allIxs []stypes.Instruction
	switch priorityLevel {
	case "medium":
		allIxs = append(allIxs, CreateSetComputeUnitPriceInstruction(10_000))
	case "high":
		allIxs = append(allIxs, CreateSetComputeUnitPriceInstruction(15_000))
	case "veryHigh":
		allIxs = append(allIxs, CreateSetComputeUnitPriceInstruction(20_000))
	}

	// convert accounts to Solana types
	var accountsSolana []stypes.AccountMeta
	for _, a := range accounts {
		accountsSolana = append(accountsSolana, stypes.AccountMeta{
			PubKey:     cmnsol.PublicKeyFromBytes(a.PubKey[:]),
			IsSigner:   a.IsSigner,
			IsWritable: a.IsWritable,
		})
	}

	// main program ix
	fillIx := stypes.Instruction{
		ProgramID: cmnsol.PublicKeyFromBytes(programKey[:]),
		Accounts:  accountsSolana,
		Data:      methodParams,
	}
	allIxs = append(allIxs, fillIx)

	// create message
	messageToSign := stypes.NewMessage(stypes.NewMessageParam{
		FeePayer:        signer,
		Instructions:    allIxs,
		RecentBlockhash: latestBlockHash,
	})

	// serialize
	txToSignBytes, err := messageToSign.Serialize()
	if err != nil {
		return nil, fmt.Errorf("can't serialize solana transaction")
	}

	// bytes32 id
	txId := crypto.Keccak256Hash(txToSignBytes)

	// persist tx bytes
	owner := contract.Address()
	key := append([]byte("solanatx:"), txId.Bytes()...)
	putBytes(evm, owner, key, txToSignBytes)

	// append to index for pagination
	appendTxIndex(evm, owner, txId)

	ev, ok := p.ABI.Events["SolanaTxGenerated"]
	if ok {
		// topics: [event sig, programKey, signer, txId] (all bytes32)
		topics := []common.Hash{
			ev.ID,
			common.BytesToHash(programKey[:]),
			common.BytesToHash(signer[:]),
			txId,
		}

		// pack ONLY non-indexed args in the same order as in the event:
		// priorityLevel, methodParams, accounts, latestBlockHash, txToSignBytes
		nonIdx := ev.Inputs.NonIndexed()
		data, perr := nonIdx.Pack(priorityLevel, methodParams, accounts, latestBlockHash, txToSignBytes)
		if perr == nil {
			evm.StateDB.AddLog(&gethtypes.Log{
				Address: contract.Address(),
				Topics:  topics,
				Data:    data,
			})
		}
	}

	// return (bytes, bytes32)
	return method.Outputs.Pack(txToSignBytes, txId)
}

func (p Precompile) getSolanaTx(
	evm *vm.EVM,
	contract *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 1, len(args))
	}
	txId := args[0].([32]byte)

	owner := contract.Address()
	key := append([]byte("solanatx:"), txId[:]...)
	txToSignBytes := getBytes(evm, owner, key)

	return method.Outputs.Pack(txToSignBytes)
}

func (p Precompile) getSolanaTxs(
	evm *vm.EVM,
	contract *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 2, len(args))
	}

	// pagination
	skipBI := args[0].(*big.Int)
	limitBI := args[1].(*big.Int)
	if skipBI.Sign() < 0 || limitBI.Sign() < 0 {
		return nil, fmt.Errorf("skip/limit must be non-negative")
	}
	skip := skipBI.Uint64()
	limit := limitBI.Uint64()
	if limit == 0 {
		return method.Outputs.Pack([][]byte{})
	}

	owner := contract.Address()
	st := evm.StateDB

	indexBase := crypto.Keccak256Hash([]byte("solanatx:index"))
	lenSlot := crypto.Keccak256Hash(append(indexBase[:], []byte("len")...))
	total := new(big.Int).SetBytes(st.GetState(owner, lenSlot).Bytes()).Uint64()
	if total == 0 || skip >= total {
		return method.Outputs.Pack([][]byte{})
	}

	start := skip
	end := start + limit
	if end > total {
		end = total
	}

	txs := make([][]byte, 0, end-start)
	for i := start; i < end; i++ {
		txIdHash := st.GetState(owner, slotForIndex(indexBase, i))
		if (txIdHash == common.Hash{}) {
			continue
		}
		key := append([]byte("solanatx:"), txIdHash.Bytes()...)
		txBytes := getBytes(evm, owner, key)
		txs = append(txs, txBytes)
	}

	return method.Outputs.Pack(txs)
}

// ABI tuple for accounts
type AccountMetaAbi struct {
	PubKey     [32]byte `abi:"pubKey"`
	IsSigner   bool     `abi:"isSigner"`
	IsWritable bool     `abi:"isWritable"`
}

func CreateSetComputeUnitPriceInstruction(mLamports uint64) stypes.Instruction {
	return compute_budget.SetComputeUnitPrice(
		compute_budget.SetComputeUnitPriceParam{MicroLamports: mLamports},
	)
}

func slotForIndex(base common.Hash, index uint64) common.Hash {
	var idx [32]byte
	binary.BigEndian.PutUint64(idx[24:], index) // big-endian u256
	return crypto.Keccak256Hash(append(base[:], idx[:]...))
}

func putBytes(evm *vm.EVM, owner common.Address, baseKey []byte, value []byte) {
	st := evm.StateDB
	base := crypto.Keccak256Hash(baseKey)
	lenSlot := crypto.Keccak256Hash(append(base[:], []byte("len")...))

	lenU256 := new(big.Int).SetUint64(uint64(len(value)))
	st.SetState(owner, lenSlot, common.BigToHash(lenU256))

	var i uint64
	for off := 0; off < len(value); off, i = off+32, i+1 {
		end := off + 32
		if end > len(value) {
			end = len(value)
		}
		var word [32]byte
		copy(word[:], value[off:end])
		st.SetState(owner, slotForIndex(base, i), common.BytesToHash(word[:]))
	}
}

func getBytes(evm *vm.EVM, owner common.Address, baseKey []byte) []byte {
	st := evm.StateDB
	base := crypto.Keccak256Hash(baseKey)
	lenSlot := crypto.Keccak256Hash(append(base[:], []byte("len")...))

	total := new(big.Int).SetBytes(st.GetState(owner, lenSlot).Bytes()).Uint64()
	if total == 0 {
		return nil
	}
	out := make([]byte, total)

	var i uint64
	var off int
	for off < int(total) {
		word := st.GetState(owner, slotForIndex(base, i)).Bytes()
		cpy := 32
		if off+cpy > int(total) {
			cpy = int(total) - off
		}
		copy(out[off:off+cpy], word[:cpy])
		off += cpy
		i++
	}
	return out
}

func appendTxIndex(evm *vm.EVM, owner common.Address, txId common.Hash) {
	st := evm.StateDB
	indexBase := crypto.Keccak256Hash([]byte("solanatx:index"))
	lenSlot := crypto.Keccak256Hash(append(indexBase[:], []byte("len")...))

	cur := new(big.Int).SetBytes(st.GetState(owner, lenSlot).Bytes()).Uint64()
	st.SetState(owner, slotForIndex(indexBase, cur), txId)
	st.SetState(owner, lenSlot, common.BigToHash(new(big.Int).SetUint64(cur+1)))
}

// IsTransaction checks if the given method name corresponds to a transaction or query.
func (Precompile) IsTransaction(method *abi.Method) bool {
	switch method.Name {
	case generateSolanaTx:
		return true
	default:
		return false
	}
}
