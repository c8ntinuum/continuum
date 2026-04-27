//go:build test

package evmd

import evmtypes "github.com/cosmos/evm/x/vm/types"

func valRewardsIntegrationRuntimeEnabled() bool {
	return true
}

func resetValRewardsIntegrationRuntime() {
	evmtypes.NewEVMConfigurator().ResetTestConfig()
}
