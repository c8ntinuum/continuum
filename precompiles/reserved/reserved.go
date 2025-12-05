package reserved

import (
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

type Precompile struct {
	no int
}

var _ vm.PrecompiledContract = &Precompile{}

// NewPrecompile creates a new empty Precompile instance as a PrecompiledContract interface.
func NewPrecompile(no int) (*Precompile, error) {
	return &Precompile{
		no: no,
	}, nil
}

// Address defines the address of the empty precompiled contract.
func (p Precompile) Address() common.Address {
	if p.no == 15 {
		return common.HexToAddress(evmtypes.ReservedSlot15PrecompileAddress)
	}
	if p.no == 16 {
		return common.HexToAddress(evmtypes.ReservedSlot16PrecompileAddress)
	}
	if p.no == 17 {
		return common.HexToAddress(evmtypes.ReservedSlot17PrecompileAddress)
	}
	if p.no == 18 {
		return common.HexToAddress(evmtypes.ReservedSlot18PrecompileAddress)
	}
	if p.no == 19 {
		return common.HexToAddress(evmtypes.ReservedSlot19PrecompileAddress)
	}
	if p.no == 20 {
		return common.HexToAddress(evmtypes.ReservedSlot20PrecompileAddress)
	}
	if p.no == 21 {
		return common.HexToAddress(evmtypes.ReservedSlot21PrecompileAddress)
	}
	if p.no == 22 {
		return common.HexToAddress(evmtypes.ReservedSlot22PrecompileAddress)
	}
	if p.no == 23 {
		return common.HexToAddress(evmtypes.ReservedSlot23PrecompileAddress)
	}
	if p.no == 24 {
		return common.HexToAddress(evmtypes.ReservedSlot24PrecompileAddress)
	}
	if p.no == 25 {
		return common.HexToAddress(evmtypes.ReservedSlot25PrecompileAddress)
	}
	if p.no == 26 {
		return common.HexToAddress(evmtypes.ReservedSlot26PrecompileAddress)
	}
	if p.no == 27 {
		return common.HexToAddress(evmtypes.ReservedSlot27PrecompileAddress)
	}
	if p.no == 28 {
		return common.HexToAddress(evmtypes.ReservedSlot28PrecompileAddress)
	}
	if p.no == 29 {
		return common.HexToAddress(evmtypes.ReservedSlot29PrecompileAddress)
	}
	if p.no == 30 {
		return common.HexToAddress(evmtypes.ReservedSlot30PrecompileAddress)
	}
	if p.no == 31 {
		return common.HexToAddress(evmtypes.ReservedSlot31PrecompileAddress)
	}
	if p.no == 32 {
		return common.HexToAddress(evmtypes.ReservedSlot32PrecompileAddress)
	}
	if p.no == 33 {
		return common.HexToAddress(evmtypes.ReservedSlot33PrecompileAddress)
	}
	if p.no == 34 {
		return common.HexToAddress(evmtypes.ReservedSlot34PrecompileAddress)
	}
	if p.no == 35 {
		return common.HexToAddress(evmtypes.ReservedSlot35PrecompileAddress)
	}
	if p.no == 36 {
		return common.HexToAddress(evmtypes.ReservedSlot36PrecompileAddress)
	}
	if p.no == 37 {
		return common.HexToAddress(evmtypes.ReservedSlot37PrecompileAddress)
	}
	if p.no == 38 {
		return common.HexToAddress(evmtypes.ReservedSlot38PrecompileAddress)
	}
	if p.no == 39 {
		return common.HexToAddress(evmtypes.ReservedSlot39PrecompileAddress)
	}
	if p.no == 40 {
		return common.HexToAddress(evmtypes.ReservedSlot40PrecompileAddress)
	}
	if p.no == 41 {
		return common.HexToAddress(evmtypes.ReservedSlot41PrecompileAddress)
	}
	if p.no == 42 {
		return common.HexToAddress(evmtypes.ReservedSlot42PrecompileAddress)
	}
	if p.no == 43 {
		return common.HexToAddress(evmtypes.ReservedSlot43PrecompileAddress)
	}
	if p.no == 44 {
		return common.HexToAddress(evmtypes.ReservedSlot44PrecompileAddress)
	}
	if p.no == 45 {
		return common.HexToAddress(evmtypes.ReservedSlot45PrecompileAddress)
	}
	if p.no == 46 {
		return common.HexToAddress(evmtypes.ReservedSlot46PrecompileAddress)
	}
	if p.no == 48 {
		return common.HexToAddress(evmtypes.ReservedSlot47PrecompileAddress)
	}
	if p.no == 48 {
		return common.HexToAddress(evmtypes.ReservedSlot48PrecompileAddress)
	}
	if p.no == 49 {
		return common.HexToAddress(evmtypes.ReservedSlot49PrecompileAddress)
	}
	if p.no == 50 {
		return common.HexToAddress(evmtypes.ReservedSlot50PrecompileAddress)
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
