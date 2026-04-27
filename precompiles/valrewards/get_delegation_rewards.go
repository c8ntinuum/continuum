package valrewards

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	cmn "github.com/cosmos/evm/precompiles/common"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

func (p Precompile) DelegationRewards(
	ctx sdk.Context,
	_ *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {

	// Validate args
	if len(args) != 2 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 2, len(args))
	}

	// Validate delegator
	delegatorAddress, ok := args[0].(common.Address)
	if !ok || delegatorAddress == (common.Address{}) {
		return nil, fmt.Errorf(cmn.ErrInvalidDelegator, args[0])
	}

	// Validate epoch
	epoch, ok := args[1].(uint64)
	if !ok {
		return nil, fmt.Errorf(cmn.ErrInvalidType, "epoch", uint64(0), args[1])
	}

	delegatorOutstandingRewardsPerEpoch, err := p.valrewardsKeeper.GetDelegationRewards(ctx, sdk.AccAddress(delegatorAddress.Bytes()), epoch)
	if err != nil {
		return nil, err
	}

	// Return coin response
	return method.Outputs.Pack(toCoinResponse(delegatorOutstandingRewardsPerEpoch))
}
