package keeper

import (
	"fmt"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/cosmos/evm/x/vm/types"
)

// WithStaticPrecompiles sets the available static precompiled contracts.
func (k *Keeper) WithStaticPrecompiles(precompiles map[common.Address]vm.PrecompiledContract) *Keeper {
	if k.precompiles != nil {
		panic("available precompiles map already set") //nolint:halt // One-time registration invariant: static precompiles must only be installed once.
	}

	if len(precompiles) == 0 {
		panic("empty precompiled contract map") //nolint:halt // Boot invariant: static precompile registration must be non-empty.
	}
	if _, found := precompiles[common.Address{}]; found {
		panic("zero address cannot be registered as a static precompile") //nolint:halt // Boot invariant: zero-address precompiles are never valid.
	}

	k.precompiles = precompiles
	return k
}

// WithStaticPrecompiles sets the available static precompiled contracts.
func (k *Keeper) RegisterStaticPrecompile(address common.Address, precompile vm.PrecompiledContract) {
	if k.precompiles == nil {
		k.precompiles = make(map[common.Address]vm.PrecompiledContract)
	}
	k.precompiles[address] = precompile
}

// GetStaticPrecompileInstance returns the instance of the given static precompile address.
func (k *Keeper) GetStaticPrecompileInstance(params *types.Params, address common.Address) (vm.PrecompiledContract, bool, error) {
	if k.IsAvailableStaticPrecompile(params, address) {
		precompile, found := k.precompiles[address]
		if !found {
			return nil, false, fmt.Errorf("%w: static precompile enabled but not registered: %s", vm.ErrExecutionReverted, address)
		}
		return precompile, true, nil
	}
	return nil, false, nil
}

// IsAvailablePrecompile returns true if the given static precompile address is contained in the
// EVM keeper's available precompiles map.
// This function assumes that the Berlin precompiles cannot be disabled.
func (k Keeper) IsAvailableStaticPrecompile(params *types.Params, address common.Address) bool {
	return slices.Contains(params.ActiveStaticPrecompiles, address.String()) ||
		slices.Contains(vm.PrecompiledAddressesPrague, address)
}
