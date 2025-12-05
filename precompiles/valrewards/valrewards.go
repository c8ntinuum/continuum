package valrewards

import (
	"bytes"
	_ "embed"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	accountkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	vrkeeper "github.com/cosmos/evm/x/valrewards/keeper"

	cmn "github.com/cosmos/evm/precompiles/common"

	storetypes "cosmossdk.io/store/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

const (
	ClaimRewardsMethod                = "claimRewards"
	DepositValidatorRewardsPoolMethod = "depositValidatorRewardsPool"
	ValidatorOutstandingRewardsMethod = "validatorOutstandingRewards"
	DelegationRewardsMethod           = "delegationRewards"
	RewardsPoolMethod                 = "rewardsPool"
)

var _ vm.PrecompiledContract = &Precompile{}

var (
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

// Precompile defines the precompiled contract for validator rewards.
type Precompile struct {
	cmn.Precompile

	abi.ABI
	accountKeeper    accountkeeper.AccountKeeper
	stakingKeeper    cmn.StakingKeeper
	bankKeeper       cmn.BankKeeper
	valrewardsKeeper vrkeeper.Keeper
}

// NewPrecompile creates a new distribution Precompile instance as a PrecompiledContract interface.
func NewPrecompile(
	accountKeeper accountkeeper.AccountKeeper,
	stakingKeeper cmn.StakingKeeper,
	bankKeeper cmn.BankKeeper,
	valrewardsKeeper vrkeeper.Keeper,
) *Precompile {
	return &Precompile{
		Precompile: cmn.Precompile{
			KvGasConfig:           storetypes.KVGasConfig(),
			TransientKVGasConfig:  storetypes.TransientGasConfig(),
			ContractAddress:       common.HexToAddress(evmtypes.ValRewardsPrecompileAddress),
			BalanceHandlerFactory: cmn.NewBalanceHandlerFactory(bankKeeper),
		},
		ABI:              ABI,
		accountKeeper:    accountKeeper,
		stakingKeeper:    stakingKeeper,
		bankKeeper:       bankKeeper,
		valrewardsKeeper: valrewardsKeeper,
	}
}

func (Precompile) Address() common.Address {
	return common.HexToAddress(evmtypes.ValRewardsPrecompileAddress)
}

// RequiredGas calculates the precompiled contract gas rate.
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
		return p.Execute(ctx, evm.StateDB, contract, readonly)
	})
}

func (p Precompile) Execute(ctx sdk.Context, stateDB vm.StateDB, contract *vm.Contract, readOnly bool) ([]byte, error) {
	method, args, err := cmn.SetupABI(p.ABI, contract, readOnly, p.IsTransaction)
	if err != nil {
		return nil, err
	}
	var bz []byte
	switch method.Name {
	case ClaimRewardsMethod:
		bz, err = p.ClaimRewards(ctx, contract, stateDB, method, args)
	case DepositValidatorRewardsPoolMethod:
		bz, err = p.DepositValidatorRewardsPool(ctx, contract, stateDB, method, args)
	case ValidatorOutstandingRewardsMethod:
		bz, err = p.ValidatorOutstandingRewards(ctx, contract, method, args)
	case DelegationRewardsMethod:
		bz, err = p.DelegationRewards(ctx, contract, method, args)
	case RewardsPoolMethod:
		bz, err = p.RewardsPool(ctx, contract, method, args)
	default:
		return nil, fmt.Errorf(cmn.ErrUnknownMethod, method.Name)
	}

	return bz, err
}

func (Precompile) IsTransaction(method *abi.Method) bool {
	switch method.Name {
	case ClaimRewardsMethod,
		DepositValidatorRewardsPoolMethod:
		return true
	default:
		return false
	}
}
