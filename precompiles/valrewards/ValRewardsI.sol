// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev The ValRewardsI contract's address.
address constant VALREWARDS_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000714;

/// @dev The ValRewardsI contract's instance.
ValRewardsI constant VALREWARDS_CONTRACT = ValRewardsI(
    VALREWARDS_PRECOMPILE_ADDRESS
);

/// @dev Coin is a struct that represents a token with a denomination and an amount.
struct Coin {
    string denom;
    uint256 amount;
}

/// @author c8ntinuum Team
/// @title Validator Rewards Precompile Contract
/// @dev The interface through which solidity contracts will interact with Validator Rewards
/// @custom:address 0x0000000000000000000000000000000000000714
interface ValRewardsI {
    /// @dev Claims all rewards from a select set of validators or all of them for a delegator.
    /// @param delegatorAddress The address of the delegator
    /// @param maxRetrieve The maximum number of validators to claim rewards from
    /// @param epoch The epoch for which rewards are claimed
    /// @return success Whether the transaction was successful or not
    function claimRewards(
        address delegatorAddress,
        uint32 maxRetrieve,
        uint64 epoch
    ) external returns (bool success);

    /// @dev depositValidatorRewardsPool defines a method to allow an account to directly
    /// fund the validator rewards pool.
    /// @param depositor The address of the depositor
    /// @param amount The amount of coin sent to the validator rewards pool
    /// @return success Whether the transaction was successful or not
    function depositValidatorRewardsPool(
        address depositor,
        Coin memory amount
    ) external returns (bool success);

    /// @dev Queries the outstanding rewards of a validator address.
    /// @param epoch The epoch for which rewards are checked
    /// @param validatorAddress The address of the validator
    /// @return rewards The validator's outstanding rewards
    function validatorOutstandingRewards(
        uint64 epoch,
        string memory validatorAddress
    ) external view returns (Coin calldata rewards);

    /// @dev Queries the total rewards accrued by a delegation from a specific epoch.
    /// @param delegatorAddress The address of the delegator
    /// @param epoch The epoch for which rewards are checked
    /// @return rewards The total rewards accrued by a delegation.
    function delegationRewards(
        address delegatorAddress,
        uint64 epoch
    ) external view returns (Coin calldata rewards);

    /// @dev Queries the total rewards in the rewards pool.
    /// @return rewards The the total rewards in the rewards pool
    function rewardsPool() external view returns (Coin calldata rewards);
}
