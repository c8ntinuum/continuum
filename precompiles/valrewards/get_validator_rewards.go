package valrewards

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	cmn "github.com/cosmos/evm/precompiles/common"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/vm"
)

func (p Precompile) ValidatorOutstandingRewards(
	ctx sdk.Context,
	_ *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {

	// Validate args
	if len(args) != 2 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 2, len(args))
	}

	// Validate epoch
	epoch, ok := args[0].(uint64)
	if !ok {
		return nil, fmt.Errorf(cmn.ErrInvalidType, "epoch", uint64(0), args[0])
	}

	// Validator address
	validatorAddress, _ := args[1].(string)
	if len([]byte(validatorAddress)) > 128 {
		return nil, fmt.Errorf(cmn.ErrInvalidArgument, "validatorAddress", args[1])
	}

	// Get validator outstanding rewards per epoch
	valOutstandingRewardsPerEpoch := p.valrewardsKeeper.GetValidatorOutstandingReward(ctx, epoch, validatorAddress)

	// Return response
	return method.Outputs.Pack(cmn.NewCoinResponse(valOutstandingRewardsPerEpoch))
}
