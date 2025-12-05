package valrewards

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cmn "github.com/cosmos/evm/precompiles/common"
	vrtypes "github.com/cosmos/evm/x/valrewards/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
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

	// Validate args
	if len(args) != 3 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 3, len(args))
	}

	// Validate delegator
	delegatorAddress, ok := args[0].(common.Address)
	if !ok || delegatorAddress == (common.Address{}) {
		return nil, fmt.Errorf(cmn.ErrInvalidDelegator, args[0])
	}
	msgSender := contract.Caller()
	if msgSender != delegatorAddress {
		return nil, fmt.Errorf(cmn.ErrRequesterIsNotMsgSender, msgSender.String(), delegatorAddress.String())
	}

	// Validate max validators to retrieve
	maxRetrieve, ok := args[1].(uint32)
	if !ok {
		return nil, fmt.Errorf(cmn.ErrInvalidType, "maxRetrieve", uint32(0), args[1])
	}
	maxVals, err := p.stakingKeeper.MaxValidators(ctx)
	if err != nil {
		return nil, err
	}
	if maxRetrieve > maxVals {
		return nil, fmt.Errorf("maxRetrieve (%d) parameter exceeds the maximum number of validators (%d)", maxRetrieve, maxVals)
	}

	// Validate epoch
	epoch, ok := args[2].(uint64)
	if !ok {
		return nil, fmt.Errorf(cmn.ErrInvalidType, "epoch", uint64(0), args[2])
	}

	// Get validators of delegator
	validators, err := p.stakingKeeper.GetDelegatorValidators(ctx, delegatorAddress.Bytes(), maxRetrieve)
	if err != nil {
		return nil, err
	}

	// Init response
	delegatorOutstandingRewardsPerEpoch := sdk.Coin{
		Denom:  evmtypes.DefaultEVMDenom,
		Amount: math.NewInt(0),
	}

	// Sum delegator validators outstanding balance
	for _, validator := range validators.Validators {
		valOutstandingRewardsPerEpoch := p.valrewardsKeeper.GetValidatorOutstandingReward(ctx, epoch, validator.OperatorAddress)
		delegatorOutstandingRewardsPerEpoch = delegatorOutstandingRewardsPerEpoch.Add(valOutstandingRewardsPerEpoch)
	}

	// Check if module balance is enough for rewards
	if delegatorOutstandingRewardsPerEpoch.IsZero() {
		return nil, fmt.Errorf(cmn.ErrNoOutstandingBalance)
	}

	// Get module balance
	moduleAddr := p.accountKeeper.GetModuleAddress(vrtypes.ModuleName)
	moduleBalance := p.bankKeeper.GetBalance(ctx, moduleAddr, evmtypes.DefaultEVMDenom)

	// Check if module balance is enough for rewards
	if moduleBalance.IsLT(delegatorOutstandingRewardsPerEpoch) {
		return nil, fmt.Errorf(cmn.ErrInsufficientRewardsBalance)
	}

	// Cast amount
	transferCoins := sdk.NewCoins(delegatorOutstandingRewardsPerEpoch)

	// Update outstanding balances
	for _, validator := range validators.Validators {
		p.valrewardsKeeper.SetValidatorOutstandingReward(ctx, epoch, validator.OperatorAddress, sdk.NewCoin(evmtypes.DefaultEVMDenom, math.ZeroInt()))
	}

	// Transfer rewards
	if err := p.bankKeeper.SendCoinsFromModuleToAccount(ctx, vrtypes.ModuleName, delegatorAddress.Bytes(), transferCoins); err != nil {
		return nil, err
	}

	// Return response
	return method.Outputs.Pack(true)
}
