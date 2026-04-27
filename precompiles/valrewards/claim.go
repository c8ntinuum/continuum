package valrewards

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	cmn "github.com/cosmos/evm/precompiles/common"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

func (p *Precompile) ClaimRewards(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	_ = contract
	_ = stateDB

	// Validate args
	if len(args) != 2 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 2, len(args))
	}

	// Validate the target validator operator account. Any EVM caller may
	// trigger this claim, but the keeper still rejects non-validator targets.
	validatorOperatorAddress, ok := args[0].(common.Address)
	if !ok || validatorOperatorAddress == (common.Address{}) {
		return nil, fmt.Errorf(cmn.ErrInvalidValidatorOperator, args[0])
	}

	// Validate epoch
	epoch, ok := args[1].(uint64)
	if !ok {
		return nil, fmt.Errorf(cmn.ErrInvalidType, "epoch", uint64(0), args[1])
	}

	_, err := p.valrewardsKeeper.ClaimOperatorRewards(ctx, sdk.AccAddress(validatorOperatorAddress.Bytes()), epoch)
	if err != nil {
		return nil, err
	}

	// Return response
	return method.Outputs.Pack(true)
}
