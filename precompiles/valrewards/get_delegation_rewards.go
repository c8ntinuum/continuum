package valrewards

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cmn "github.com/cosmos/evm/precompiles/common"
	evmtypes "github.com/cosmos/evm/x/vm/types"
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

	// Get validators of delegator
	maxVals, err := p.stakingKeeper.MaxValidators(ctx)
	if err != nil {
		return nil, err
	}
	validators, err := p.stakingKeeper.GetDelegatorValidators(ctx, delegatorAddress.Bytes(), maxVals)
	if err != nil {
		return nil, err
	}

	// Init response
	delegatorOutstandingRewardsPerEpoch := sdk.Coin{
		Denom:  evmtypes.DefaultEVMDenom,
		Amount: math.NewInt(0),
	}

	// Iterate all delegators validators and add outstanding reward
	for _, validator := range validators.Validators {
		valOutstandingRewardsPerEpoch := p.valrewardsKeeper.GetValidatorOutstandingReward(ctx, epoch, validator.OperatorAddress)
		delegatorOutstandingRewardsPerEpoch = delegatorOutstandingRewardsPerEpoch.Add(valOutstandingRewardsPerEpoch)
	}

	// Return coin response
	return method.Outputs.Pack(cmn.NewCoinResponse(delegatorOutstandingRewardsPerEpoch))
}
