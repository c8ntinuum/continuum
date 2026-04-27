package evmd

import (
	"encoding/json"
	"fmt"

	sdkmath "cosmossdk.io/math"
	feegrant "cosmossdk.io/x/feegrant"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	evmconfig "github.com/cosmos/evm/config"
	evmutils "github.com/cosmos/evm/utils"
	valrewardstypes "github.com/cosmos/evm/x/valrewards/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	proto "github.com/cosmos/gogoproto/proto"
)

// CrossGenesisValidate enforces cross-module genesis invariants that are not
// covered by module-local ValidateGenesis implementations.
func CrossGenesisValidate(cdc codec.Codec, genState GenesisState) error {
	return CrossGenesisValidateAtInitialHeight(cdc, genState, 1)
}

// CrossGenesisValidateAtInitialHeight enforces cross-module genesis invariants
// that depend on the chain's effective start height.
func CrossGenesisValidateAtInitialHeight(cdc codec.Codec, genState GenesisState, initialHeight int64) error {
	initialHeight = normalizeGenesisInitialHeight(initialHeight)

	unpacker, ok := cdc.(codectypes.AnyUnpacker)
	if !ok {
		return fmt.Errorf("codec %T does not implement AnyUnpacker", cdc)
	}

	var authGen authtypes.GenesisState
	if err := unmarshalModuleGenesis(cdc, genState, authtypes.ModuleName, &authGen); err != nil {
		return err
	}
	if err := authGen.UnpackInterfaces(unpacker); err != nil {
		return fmt.Errorf("unpack auth genesis: %w", err)
	}
	authAccounts, err := authAccountAddressSet(authGen)
	if err != nil {
		return fmt.Errorf("build auth account set: %w", err)
	}

	moduleAccounts := moduleAccountAddressSet()

	var bankGen banktypes.GenesisState
	if err := unmarshalModuleGenesis(cdc, genState, banktypes.ModuleName, &bankGen); err != nil {
		return err
	}
	bankBalances := make(map[string]struct{}, len(bankGen.Balances))
	for _, balance := range bankGen.Balances {
		bankBalances[balance.Address] = struct{}{}

		if _, ok := authAccounts[balance.Address]; ok {
			continue
		}
		if _, ok := moduleAccounts[balance.Address]; ok {
			continue
		}

		return fmt.Errorf("bank balance address %s has no matching auth account or module account", balance.Address)
	}

	var stakingGen stakingtypes.GenesisState
	if err := unmarshalModuleGenesis(cdc, genState, stakingtypes.ModuleName, &stakingGen); err != nil {
		return err
	}
	if err := stakingGen.UnpackInterfaces(unpacker); err != nil {
		return fmt.Errorf("unpack staking genesis: %w", err)
	}

	validatorSet := make(map[string]stakingtypes.Validator, len(stakingGen.Validators))
	validatorConsAddrs := make(map[string]string, len(stakingGen.Validators))
	validatorConsAddrSet := make(map[string]struct{}, len(stakingGen.Validators))
	for _, validator := range stakingGen.Validators {
		validatorSet[validator.OperatorAddress] = validator

		consAddr, err := validator.GetConsAddr()
		if err != nil {
			return fmt.Errorf("staking validator %s has invalid consensus key: %w", validator.OperatorAddress, err)
		}
		consAddrStr := sdk.ConsAddress(consAddr).String()
		validatorConsAddrs[validator.OperatorAddress] = consAddrStr
		validatorConsAddrSet[consAddrStr] = struct{}{}
	}

	for _, redelegation := range stakingGen.Redelegations {
		if _, ok := validatorSet[redelegation.ValidatorSrcAddress]; !ok {
			return fmt.Errorf(
				"staking redelegation for delegator %s references unknown source validator %s",
				redelegation.DelegatorAddress,
				redelegation.ValidatorSrcAddress,
			)
		}
		if _, ok := validatorSet[redelegation.ValidatorDstAddress]; !ok {
			return fmt.Errorf(
				"staking redelegation for delegator %s references unknown destination validator %s",
				redelegation.DelegatorAddress,
				redelegation.ValidatorDstAddress,
			)
		}
	}

	for _, unbondingDelegation := range stakingGen.UnbondingDelegations {
		if _, ok := validatorSet[unbondingDelegation.ValidatorAddress]; !ok {
			return fmt.Errorf(
				"staking unbonding_delegation for delegator %s references unknown validator %s",
				unbondingDelegation.DelegatorAddress,
				unbondingDelegation.ValidatorAddress,
			)
		}
	}

	// Bank ↔ staking pool invariant: the bonded_tokens_pool and
	// not_bonded_tokens_pool module-account balances must equal Σ
	// validator.tokens grouped by status (plus unbonding_delegation entries
	// for the not-bonded side). This is the same invariant that
	// staking.InitGenesis panics on at cosmos-sdk/x/staking/keeper/genesis.go
	// lines 157 and 173 — surfacing it here gives `ctmd genesis validate`
	// (and the live InitChainer) a clear, named error instead of a late
	// panic buried in SDK code during boot.
	bondDenom := stakingGen.Params.BondDenom
	expectedBonded := sdkmath.ZeroInt()
	expectedNotBonded := sdkmath.ZeroInt()
	for _, validator := range stakingGen.Validators {
		switch validator.Status {
		case stakingtypes.Bonded:
			expectedBonded = expectedBonded.Add(validator.Tokens)
		case stakingtypes.Unbonding, stakingtypes.Unbonded:
			expectedNotBonded = expectedNotBonded.Add(validator.Tokens)
		}
	}
	for _, ubd := range stakingGen.UnbondingDelegations {
		for _, entry := range ubd.Entries {
			expectedNotBonded = expectedNotBonded.Add(entry.Balance)
		}
	}

	bondedPoolAddr := authtypes.NewModuleAddress(stakingtypes.BondedPoolName).String()
	notBondedPoolAddr := authtypes.NewModuleAddress(stakingtypes.NotBondedPoolName).String()

	actualBonded := sdkmath.ZeroInt()
	actualNotBonded := sdkmath.ZeroInt()
	for _, balance := range bankGen.Balances {
		switch balance.Address {
		case bondedPoolAddr:
			actualBonded = balance.Coins.AmountOf(bondDenom)
		case notBondedPoolAddr:
			actualNotBonded = balance.Coins.AmountOf(bondDenom)
		}
	}

	if !actualBonded.Equal(expectedBonded) {
		return fmt.Errorf(
			"bank bonded_tokens_pool balance %s%s does not match Σ validators[Status=Bonded].tokens %s%s (delta %s)",
			actualBonded, bondDenom, expectedBonded, bondDenom, expectedBonded.Sub(actualBonded),
		)
	}
	if !actualNotBonded.Equal(expectedNotBonded) {
		return fmt.Errorf(
			"bank not_bonded_tokens_pool balance %s%s does not match Σ validators[Status∈{Unbonding,Unbonded}].tokens + Σ unbonding_delegation entries %s%s (delta %s)",
			actualNotBonded, bondDenom, expectedNotBonded, bondDenom, expectedNotBonded.Sub(actualNotBonded),
		)
	}

	// Bank ↔ distribution invariant: the distribution module-account bank
	// balance must equal truncate(community_pool + Σ outstanding_rewards),
	// because distribution tracks those amounts as DecCoins internally while
	// bank holds only integer Coins. This is the invariant that
	// distribution.InitGenesis panics on at cosmos-sdk/x/distribution/keeper/
	// genesis.go:131 — surfacing it here gives ctmd genesis validate an
	// early, named error instead of a late panic during boot.
	{
		var distrBalGen distrtypes.GenesisState
		if err := unmarshalModuleGenesis(cdc, genState, distrtypes.ModuleName, &distrBalGen); err != nil {
			return err
		}

		moduleHoldings := sdk.DecCoins{}
		for _, rew := range distrBalGen.OutstandingRewards {
			moduleHoldings = moduleHoldings.Add(rew.OutstandingRewards...)
		}
		moduleHoldings = moduleHoldings.Add(distrBalGen.FeePool.CommunityPool...)
		expectedHoldings, _ := moduleHoldings.TruncateDecimal()

		distrModuleAddr := authtypes.NewModuleAddress(distrtypes.ModuleName).String()
		var actualHoldings sdk.Coins
		for _, balance := range bankGen.Balances {
			if balance.Address == distrModuleAddr {
				actualHoldings = balance.Coins
				break
			}
		}
		// sdk.Coins{}.Equal(sdk.Coins{}) is true, so missing-entry vs.
		// zero-expected is handled without special-casing.
		if !actualHoldings.Equal(expectedHoldings) {
			return fmt.Errorf(
				"bank %s balance %s does not match truncate(community_pool + Σ outstanding_rewards) = %s",
				distrtypes.ModuleName, actualHoldings, expectedHoldings,
			)
		}
	}

	if stakingGen.Exported {
		var distrGen distrtypes.GenesisState
		if err := unmarshalModuleGenesis(cdc, genState, distrtypes.ModuleName, &distrGen); err != nil {
			return err
		}

		historicalRewards := make(map[string]map[uint64]distrtypes.ValidatorHistoricalRewards)
		for _, record := range distrGen.ValidatorHistoricalRewards {
			if historicalRewards[record.ValidatorAddress] == nil {
				historicalRewards[record.ValidatorAddress] = make(map[uint64]distrtypes.ValidatorHistoricalRewards)
			}
			historicalRewards[record.ValidatorAddress][record.Period] = record.Rewards
		}

		currentRewards := make(map[string]distrtypes.ValidatorCurrentRewards, len(distrGen.ValidatorCurrentRewards))
		for _, record := range distrGen.ValidatorCurrentRewards {
			if record.Rewards.Period == 0 {
				return fmt.Errorf("distribution current rewards for validator %s has invalid period 0", record.ValidatorAddress)
			}
			currentRewards[record.ValidatorAddress] = record.Rewards
		}

		accumulatedCommissions := make(map[string]struct{}, len(distrGen.ValidatorAccumulatedCommissions))
		for _, record := range distrGen.ValidatorAccumulatedCommissions {
			accumulatedCommissions[record.ValidatorAddress] = struct{}{}
		}

		outstandingRewards := make(map[string]struct{}, len(distrGen.OutstandingRewards))
		for _, record := range distrGen.OutstandingRewards {
			outstandingRewards[record.ValidatorAddress] = struct{}{}
		}

		delegatorStartingInfos := make(map[string]struct{}, len(distrGen.DelegatorStartingInfos))
		requiredHistoricalRefs := make(map[string]map[uint64]uint32)
		bumpHistoricalRef := func(validatorAddr string, period uint64) {
			if requiredHistoricalRefs[validatorAddr] == nil {
				requiredHistoricalRefs[validatorAddr] = make(map[uint64]uint32)
			}
			requiredHistoricalRefs[validatorAddr][period]++
		}

		for _, record := range distrGen.DelegatorStartingInfos {
			current, ok := currentRewards[record.ValidatorAddress]
			if ok && record.StartingInfo.PreviousPeriod >= current.Period {
				return fmt.Errorf(
					"distribution delegator starting info for delegator %s and validator %s has previous period %d, expected less than current period %d",
					record.DelegatorAddress,
					record.ValidatorAddress,
					record.StartingInfo.PreviousPeriod,
					current.Period,
				)
			}

			delegatorStartingInfos[delegatorStartingInfoKey(record.DelegatorAddress, record.ValidatorAddress)] = struct{}{}
			bumpHistoricalRef(record.ValidatorAddress, record.StartingInfo.PreviousPeriod)
		}

		for _, record := range distrGen.ValidatorSlashEvents {
			if record.Period == 0 {
				return fmt.Errorf("distribution slash event for validator %s at height %d has invalid period 0", record.ValidatorAddress, record.Height)
			}
			if record.ValidatorSlashEvent.ValidatorPeriod == 0 {
				return fmt.Errorf("distribution slash event for validator %s at height %d has invalid validator period 0", record.ValidatorAddress, record.Height)
			}
			if record.Period != record.ValidatorSlashEvent.ValidatorPeriod {
				return fmt.Errorf(
					"distribution slash event for validator %s at height %d has mismatched period %d and validator period %d",
					record.ValidatorAddress,
					record.Height,
					record.Period,
					record.ValidatorSlashEvent.ValidatorPeriod,
				)
			}
			current, ok := currentRewards[record.ValidatorAddress]
			if ok && record.Period >= current.Period {
				return fmt.Errorf(
					"distribution slash event for validator %s at height %d has period %d, expected less than current period %d",
					record.ValidatorAddress,
					record.Height,
					record.Period,
					current.Period,
				)
			}
			bumpHistoricalRef(record.ValidatorAddress, record.Period)
		}

		var slashingGen slashingtypes.GenesisState
		if err := unmarshalModuleGenesis(cdc, genState, slashingtypes.ModuleName, &slashingGen); err != nil {
			return err
		}

		signingInfos := make(map[string]struct{}, len(slashingGen.SigningInfos))
		for _, info := range slashingGen.SigningInfos {
			signingInfos[info.Address] = struct{}{}
		}

		missedBlocks := make(map[string]struct{}, len(slashingGen.MissedBlocks))
		for _, entry := range slashingGen.MissedBlocks {
			missedBlocks[entry.Address] = struct{}{}
		}
		for _, entry := range slashingGen.MissedBlocks {
			if _, ok := validatorConsAddrSet[entry.Address]; !ok {
				return fmt.Errorf("slashing missed_blocks entry %s has no matching validator", entry.Address)
			}
			if _, ok := signingInfos[entry.Address]; !ok {
				return fmt.Errorf("slashing missed_blocks entry %s has no matching signing_info", entry.Address)
			}
		}
		for _, info := range slashingGen.SigningInfos {
			if _, ok := validatorConsAddrSet[info.Address]; !ok {
				return fmt.Errorf("slashing signing_info entry %s has no matching validator", info.Address)
			}
		}

		for validatorAddr := range validatorSet {
			current, ok := currentRewards[validatorAddr]
			if !ok {
				return fmt.Errorf("distribution missing current rewards for validator %s", validatorAddr)
			}
			bumpHistoricalRef(validatorAddr, current.Period-1)

			if _, ok := accumulatedCommissions[validatorAddr]; !ok {
				return fmt.Errorf("distribution missing accumulated commission for validator %s", validatorAddr)
			}

			if _, ok := outstandingRewards[validatorAddr]; !ok {
				return fmt.Errorf("distribution missing outstanding rewards for validator %s", validatorAddr)
			}

			consAddr := validatorConsAddrs[validatorAddr]
			if _, ok := signingInfos[consAddr]; !ok {
				return fmt.Errorf("slashing missing signing info for validator %s (%s)", validatorAddr, consAddr)
			}
			if _, ok := missedBlocks[consAddr]; !ok {
				return fmt.Errorf("slashing missing missed blocks row for validator %s (%s)", validatorAddr, consAddr)
			}

			for period, expectedRefs := range requiredHistoricalRefs[validatorAddr] {
				rewardsByPeriod := historicalRewards[validatorAddr]
				if rewardsByPeriod == nil {
					return fmt.Errorf("distribution missing historical rewards period %d for validator %s", period, validatorAddr)
				}

				historical, ok := rewardsByPeriod[period]
				if !ok {
					return fmt.Errorf("distribution missing historical rewards period %d for validator %s", period, validatorAddr)
				}
				if historical.ReferenceCount < expectedRefs {
					return fmt.Errorf(
						"distribution historical rewards period %d for validator %s has reference count %d, expected at least %d",
						period,
						validatorAddr,
						historical.ReferenceCount,
						expectedRefs,
					)
				}
			}
		}

		for _, delegation := range stakingGen.Delegations {
			key := delegatorStartingInfoKey(delegation.DelegatorAddress, delegation.ValidatorAddress)
			if _, ok := delegatorStartingInfos[key]; !ok {
				return fmt.Errorf(
					"distribution missing delegator starting info for delegator %s and validator %s",
					delegation.DelegatorAddress,
					delegation.ValidatorAddress,
				)
			}
		}

		lastTotalPower := sdkmath.ZeroInt()
		for _, entry := range stakingGen.LastValidatorPowers {
			if _, ok := validatorSet[entry.Address]; !ok {
				return fmt.Errorf("staking last validator power entry %s has no matching validator", entry.Address)
			}
			lastTotalPower = lastTotalPower.Add(sdkmath.NewInt(entry.Power))
		}
		if !lastTotalPower.Equal(stakingGen.LastTotalPower) {
			return fmt.Errorf(
				"staking last_total_power mismatch: got %s, expected %s from last_validator_powers",
				stakingGen.LastTotalPower,
				lastTotalPower,
			)
		}

		for _, record := range distrGen.ValidatorHistoricalRewards {
			if _, ok := validatorSet[record.ValidatorAddress]; !ok {
				return fmt.Errorf(
					"distribution historical_rewards row references unknown validator %s (period %d)",
					record.ValidatorAddress,
					record.Period,
				)
			}
		}
		for _, record := range distrGen.ValidatorCurrentRewards {
			if _, ok := validatorSet[record.ValidatorAddress]; !ok {
				return fmt.Errorf("distribution current_rewards row references unknown validator %s", record.ValidatorAddress)
			}
		}
		for _, record := range distrGen.ValidatorAccumulatedCommissions {
			if _, ok := validatorSet[record.ValidatorAddress]; !ok {
				return fmt.Errorf("distribution accumulated_commission row references unknown validator %s", record.ValidatorAddress)
			}
		}
		for _, record := range distrGen.OutstandingRewards {
			if _, ok := validatorSet[record.ValidatorAddress]; !ok {
				return fmt.Errorf("distribution outstanding_rewards row references unknown validator %s", record.ValidatorAddress)
			}
		}
		for _, record := range distrGen.DelegatorStartingInfos {
			if _, ok := validatorSet[record.ValidatorAddress]; !ok {
				return fmt.Errorf(
					"distribution delegator_starting_info for delegator %s references unknown validator %s",
					record.DelegatorAddress,
					record.ValidatorAddress,
				)
			}
		}
		for _, record := range distrGen.ValidatorSlashEvents {
			if _, ok := validatorSet[record.ValidatorAddress]; !ok {
				return fmt.Errorf(
					"distribution slash_event references unknown validator %s (height %d period %d)",
					record.ValidatorAddress,
					record.Height,
					record.Period,
				)
			}
		}
	}

	var govGen govv1.GenesisState
	if err := unmarshalModuleGenesis(cdc, genState, "gov", &govGen); err != nil {
		return err
	}

	proposals := make(map[uint64]struct{}, len(govGen.Proposals))
	for _, proposal := range govGen.Proposals {
		proposals[proposal.Id] = struct{}{}
	}
	for _, deposit := range govGen.Deposits {
		if _, ok := proposals[deposit.ProposalId]; !ok {
			return fmt.Errorf("gov deposit references missing proposal %d", deposit.ProposalId)
		}
	}
	for _, vote := range govGen.Votes {
		if _, ok := proposals[vote.ProposalId]; !ok {
			return fmt.Errorf("gov vote references missing proposal %d", vote.ProposalId)
		}
	}

	var feegrantGen feegrant.GenesisState
	if err := unmarshalModuleGenesis(cdc, genState, feegrant.ModuleName, &feegrantGen); err != nil {
		return err
	}
	for _, allowance := range feegrantGen.Allowances {
		if _, ok := authAccounts[allowance.Granter]; !ok {
			return fmt.Errorf("feegrant granter %s has no matching auth account", allowance.Granter)
		}
	}

	var vmGen evmtypes.GenesisState
	if err := unmarshalModuleGenesis(cdc, genState, evmtypes.ModuleName, &vmGen); err != nil {
		return err
	}
	for _, preinstall := range vmGen.Preinstalls {
		accAddr := evmutils.Bech32StringFromHexAddress(preinstall.Address)
		if _, ok := authAccounts[accAddr]; ok {
			return fmt.Errorf("vm preinstall %s conflicts with auth account %s", preinstall.Address, accAddr)
		}
		if _, ok := moduleAccounts[accAddr]; ok {
			return fmt.Errorf("vm preinstall %s conflicts with module account %s", preinstall.Address, accAddr)
		}
		if _, ok := bankBalances[accAddr]; ok {
			return fmt.Errorf("vm preinstall %s conflicts with bank balance address %s", preinstall.Address, accAddr)
		}
	}

	var valRewardsGen valrewardstypes.GenesisState
	if err := unmarshalModuleGenesis(cdc, genState, valrewardstypes.ModuleName, &valRewardsGen); err != nil {
		return err
	}
	for _, entry := range valRewardsGen.ValidatorPoints {
		if _, ok := validatorSet[entry.ValidatorAddress]; !ok {
			return fmt.Errorf("valrewards validator_points entry references missing staking validator %s", entry.ValidatorAddress)
		}
	}
	for _, entry := range valRewardsGen.ValidatorOutstandingRewards {
		if _, ok := validatorSet[entry.ValidatorAddress]; !ok {
			return fmt.Errorf("valrewards validator_outstanding_rewards entry references missing staking validator %s", entry.ValidatorAddress)
		}
	}

	// Reject already-due upgrade plans so imported genesis cannot coordinate an
	// upgrade halt before the new chain has advanced past its effective start height.
	var upgradeGen upgradeGenesisPlanEnvelope
	if err := unmarshalModuleGenesis(cdc, genState, upgradetypes.ModuleName, &upgradeGen); err != nil {
		return err
	}
	if upgradeGen.Plan.Name != "" || upgradeGen.Plan.Height != 0 {
		if upgradeGen.Plan.Height <= initialHeight {
			return fmt.Errorf(
				"upgrade plan %q has stale height %d (must be > start height %d)",
				upgradeGen.Plan.Name,
				upgradeGen.Plan.Height,
				initialHeight,
			)
		}
	}

	return nil
}

type upgradeGenesisPlanEnvelope struct {
	Plan upgradetypes.Plan `json:"plan"`
}

func normalizeGenesisInitialHeight(initialHeight int64) int64 {
	if initialHeight <= 0 {
		return 1
	}

	return initialHeight
}

func unmarshalModuleGenesis(cdc codec.Codec, genState GenesisState, moduleName string, target any) error {
	if genState[moduleName] == nil {
		return nil
	}

	var err error
	if msg, ok := target.(proto.Message); ok {
		err = cdc.UnmarshalJSON(genState[moduleName], msg)
	} else {
		err = json.Unmarshal(genState[moduleName], target)
	}

	if err != nil {
		return fmt.Errorf("unmarshal %s genesis: %w", moduleName, err)
	}

	return nil
}

func authAccountAddressSet(authGen authtypes.GenesisState) (map[string]struct{}, error) {
	accounts, err := authtypes.UnpackAccounts(authGen.Accounts)
	if err != nil {
		return nil, err
	}

	addresses := make(map[string]struct{}, len(accounts))
	for _, account := range accounts {
		addresses[account.GetAddress().String()] = struct{}{}
	}

	return addresses, nil
}

func moduleAccountAddressSet() map[string]struct{} {
	moduleAccounts := make(map[string]struct{})
	for moduleName := range evmconfig.GetMaccPerms() {
		moduleAccounts[authtypes.NewModuleAddress(moduleName).String()] = struct{}{}
	}
	return moduleAccounts
}

func delegatorStartingInfoKey(delegatorAddr, validatorAddr string) string {
	return delegatorAddr + "/" + validatorAddr
}
