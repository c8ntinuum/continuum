package valrewards

import (
	"fmt"

	cmn "github.com/cosmos/evm/precompiles/common"

	sdk "github.com/cosmos/cosmos-sdk/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

func (p *Precompile) DepositValidatorRewardsPool(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {

	// Validate args
	if len(args) != 2 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 2, len(args))
	}

	// Validate depositor
	depositor, ok := args[0].(common.Address)
	if !ok || depositor == (common.Address{}) {
		return nil, fmt.Errorf(cmn.ErrInvalidHexAddress, args[0])
	}
	msgSender := contract.Caller()
	if msgSender != depositor {
		return nil, fmt.Errorf(cmn.ErrRequesterIsNotMsgSender, msgSender.String(), depositor.String())
	}

	// Validate amount
	coin, err := parseCoinArg(args[1])
	if err != nil {
		return nil, fmt.Errorf(cmn.ErrInvalidAmount, "amount arg")
	}
	if coin.Denom != evmtypes.DefaultEVMDenom {
		return nil, fmt.Errorf(cmn.ErrInvalidDenom, coin.Denom)
	}

	// Cast amount
	coins, err := cmn.NewSdkCoinsFromCoins([]cmn.Coin{coin})
	if err != nil {
		return nil, err
	}

	// Deposit into validator rewards module
	if err := p.valrewardsKeeper.DepositRewardsPoolCoins(ctx, sdk.AccAddress(depositor.Bytes()), coins[0]); err != nil {
		return nil, err
	}

	// Return response
	return method.Outputs.Pack(true)
}
