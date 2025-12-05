package valrewards

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	cmn "github.com/cosmos/evm/precompiles/common"
	vrtypes "github.com/cosmos/evm/x/valrewards/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/vm"
)

func (p Precompile) RewardsPool(
	ctx sdk.Context,
	_ *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {

	// Get module balance
	moduleAddr := p.accountKeeper.GetModuleAddress(vrtypes.ModuleName)
	moduleBalance := p.bankKeeper.GetBalance(ctx, moduleAddr, evmtypes.DefaultEVMDenom)
	return method.Outputs.Pack(cmn.NewCoinResponse(moduleBalance))
}
