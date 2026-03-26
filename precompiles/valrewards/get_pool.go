package valrewards

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	moduleBalance := p.valrewardsKeeper.GetRewardsPool(ctx)
	return method.Outputs.Pack(toCoinResponse(moduleBalance))
}
