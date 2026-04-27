package reserved

import (
	"fmt"

	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

const (
	minReservedSlot = 15
	maxReservedSlot = 50
)

var reservedSlotAddresses = [...]string{
	evmtypes.ReservedSlot15PrecompileAddress,
	evmtypes.ReservedSlot16PrecompileAddress,
	evmtypes.ReservedSlot17PrecompileAddress,
	evmtypes.ReservedSlot18PrecompileAddress,
	evmtypes.ReservedSlot19PrecompileAddress,
	evmtypes.ReservedSlot20PrecompileAddress,
	evmtypes.ReservedSlot21PrecompileAddress,
	evmtypes.ReservedSlot22PrecompileAddress,
	evmtypes.ReservedSlot23PrecompileAddress,
	evmtypes.ReservedSlot24PrecompileAddress,
	evmtypes.ReservedSlot25PrecompileAddress,
	evmtypes.ReservedSlot26PrecompileAddress,
	evmtypes.ReservedSlot27PrecompileAddress,
	evmtypes.ReservedSlot28PrecompileAddress,
	evmtypes.ReservedSlot29PrecompileAddress,
	evmtypes.ReservedSlot30PrecompileAddress,
	evmtypes.ReservedSlot31PrecompileAddress,
	evmtypes.ReservedSlot32PrecompileAddress,
	evmtypes.ReservedSlot33PrecompileAddress,
	evmtypes.ReservedSlot34PrecompileAddress,
	evmtypes.ReservedSlot35PrecompileAddress,
	evmtypes.ReservedSlot36PrecompileAddress,
	evmtypes.ReservedSlot37PrecompileAddress,
	evmtypes.ReservedSlot38PrecompileAddress,
	evmtypes.ReservedSlot39PrecompileAddress,
	evmtypes.ReservedSlot40PrecompileAddress,
	evmtypes.ReservedSlot41PrecompileAddress,
	evmtypes.ReservedSlot42PrecompileAddress,
	evmtypes.ReservedSlot43PrecompileAddress,
	evmtypes.ReservedSlot44PrecompileAddress,
	evmtypes.ReservedSlot45PrecompileAddress,
	evmtypes.ReservedSlot46PrecompileAddress,
	evmtypes.ReservedSlot47PrecompileAddress,
	evmtypes.ReservedSlot48PrecompileAddress,
	evmtypes.ReservedSlot49PrecompileAddress,
	evmtypes.ReservedSlot50PrecompileAddress,
}

type Precompile struct {
	no int
}

var _ vm.PrecompiledContract = &Precompile{}

// NewPrecompile creates a new empty Precompile instance as a PrecompiledContract interface.
func NewPrecompile(no int) (*Precompile, error) {
	if _, ok := reservedAddressForSlot(no); !ok {
		return nil, fmt.Errorf("unsupported reserved precompile slot: %d", no)
	}

	return &Precompile{
		no: no,
	}, nil
}

// Address defines the address of the empty precompiled contract.
func (p Precompile) Address() common.Address {
	address, ok := reservedAddressForSlot(p.no)
	if ok {
		return address
	}

	return common.Address{}
}

// RequiredGas returns the static gas required to execute the precompiled contract.
func (p Precompile) RequiredGas(_ []byte) uint64 {
	return 0
}

func (p *Precompile) Run(_ *vm.EVM, contract *vm.Contract, _ bool) (res []byte, err error) {
	return nil, nil
}

func reservedAddressForSlot(no int) (common.Address, bool) {
	if no < minReservedSlot || no > maxReservedSlot {
		return common.Address{}, false
	}

	return common.HexToAddress(reservedSlotAddresses[no-minReservedSlot]), true
}
