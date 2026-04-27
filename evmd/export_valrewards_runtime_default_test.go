//go:build !test

package evmd

func valRewardsIntegrationRuntimeEnabled() bool {
	return false
}

func resetValRewardsIntegrationRuntime() {}
