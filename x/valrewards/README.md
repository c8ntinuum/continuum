# x/valrewards

## Purpose

`x/valrewards` awards validators for signed blocks using per-epoch points. Rewards are not share-based. Each validator earns points for signing and a proposer bonus. At epoch boundaries, rewards are allocated proportionally to points and become claimable by the validator operator.

## Core Logic

- **Points**: On each block, validators that signed receive points proportional to voting power. The block proposer receives a bonus.
- **Epoch Rewards**: At the start of a new epoch, total points are used to split a fixed reward amount across validators.
- **Claiming**: Only the validator operator can claim rewards for their validator and epoch. Claims are one-time; outstanding rewards are zeroed after a successful claim.
- **Funding**: Claims are paid from the module rewards pool. If the pool is empty, claims fail with insufficient balance.
- **Safety Checks**:
    - If total points in an epoch is zero, the epoch is advanced with no distribution.
    - If total voting power in a block is zero, no points are awarded.

## Exposed Methods

### EVM Precompile (address: `0x0000000000000000000000000000000000000714`)

- `rewardsPool()`  
  Returns the current module pool balance.
- `validatorOutstandingRewards(epoch, validatorAddress)`  
  Returns outstanding rewards for a validator operator address and epoch.
- `delegationRewards(delegatorAddress, epoch)`  
  Returns outstanding rewards for the validator operator derived from `delegatorAddress`.
  Note: this requires `delegatorAddress` to be a validator operator.
- `depositValidatorRewardsPool(depositor, amount)`  
  Deposits funds into the rewards pool.
- `claimRewards(delegatorAddress, maxRetrieve, epoch)`  
  Claims rewards for the validator operator derived from `delegatorAddress`.
  Note: this requires `delegatorAddress` to be a validator operator.

### CLI (AutoCLI)

- Query
    - `query valrewards rewards-pool`
    - `query valrewards validator-outstanding-rewards [epoch] [validator-address]`
    - `query valrewards delegation-rewards [delegator] [epoch]`
- Tx
    - `tx valrewards deposit [depositor] [amount]`
    - `tx valrewards claim [delegator] [max-retrieve] [epoch]`

## Default Genesis

`x/valrewards` does not define its own genesis parameters/state payload.
`DefaultGenesis` and `ExportGenesis` are `nil` for this module.

Practical implication:

- do not add an `app_state.valrewards` object with params.
- state is populated at runtime (points, outstanding rewards, epoch bookkeeping)
  by begin-block logic and user transactions.

## Test Scenarios

### Precompile integration

- Rewards pool query via precompile
- Validator outstanding rewards via precompile
- Delegation rewards via precompile (operator-only)
- Deposit into rewards pool via precompile
- Claim rewards via precompile and verify rewards are cleared

### CLI demo test (in-process network)

- Wait for a full epoch to accrue points
- Query rewards pool
- Query validator outstanding rewards
- Query delegation rewards (operator-only)
- Deposit rewards pool via CLI
- Claim rewards via CLI
- Verify rewards cleared after claim

## Test Commands

### Precompile integration

```bash
go test -tags=test ./tests/integration/precompiles/valrewards -run ValRewardsPrecompileIntegrationTestSuite -count=1
```

**Params**

- `-tags=test`: uses test-only epoch settings to run fast (`BLOCKS_IN_EPOCH=10`).
- `-run ValRewardsPrecompileIntegrationTestSuite`: runs only the valrewards precompile suite.
- `-count=1`: disables test caching.

### CLI demo

```bash
go test -tags=test ./tests/integration/valrewards -run ValRewardsCLIDemo -count=1
```

**Params**

- `-tags=test`: uses test-only epoch settings to run fast (`BLOCKS_IN_EPOCH=10`).
- `-run ValRewardsCLIDemo`: runs only the CLI demo test.
- `-count=1`: disables test caching.
