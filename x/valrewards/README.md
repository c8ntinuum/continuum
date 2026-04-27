# x/valrewards

## Purpose

`x/valrewards` awards validators for signed blocks using per-epoch points.
Rewards are not share-based. Each validator earns points for signing and a
fixed proposer bonus. At epoch boundaries, rewards are allocated
proportionally to points and become claimable by the validator operator.

The module now supports genesis-backed and runtime-configurable reward
settings:

- `blocks_in_epoch`
- `rewards_per_epoch`
- `rewarding_paused`

These settings are staged for the next epoch. A change made during epoch `N`
does not alter the calculations already in progress for epoch `N`. The new
values become active only when epoch `N+1` starts.

## Runtime Model

- **Current vs next settings**:
  - `current_reward_settings` are used for the epoch currently accumulating
    points.
  - `next_reward_settings` are the staged values that will become active at the
    next epoch boundary.
- **Whitelist enforcement**:
  - whitelisted accounts can submit the runtime setter messages,
  - whitelist changes themselves are governance-only through
    `MsgUpdateParams`.
- **Pause behavior**:
  - if `rewarding_paused` is switched to `true` during an epoch, the current
    epoch still completes and pays normally,
  - the next epoch starts paused and records no new validator points,
  - no new outstanding reward rows are created for a paused epoch because no
    points are accumulated.
- **Fixed proposer bonus**:
  - `PROPOSER_BONUS_POINTS` remains hardcoded and is currently `1`.

## Core Logic

- **Points**: on each block, validators that signed receive points
  proportional to voting power. The proposer receives the fixed proposer
  bonus.
- **Epoch rewards**: at the end of an epoch, total points are used to split the
  configured `rewards_per_epoch` amount across validators.
- **Claiming**: rewards are claimable only for valid validator operator
  accounts and specific epochs. Claims are one-time; outstanding rewards are
  zeroed after a successful claim. The claim path is single-validator only.
  The Cosmos msg path and the EVM precompile path are both sponsor-callable:
  any caller may trigger the claim, but the target address must be a valid
  validator operator and the payout always goes to that target operator.
- **Funding**: claims are paid from the module rewards pool. If the pool is
  empty, claims fail with insufficient balance.
- **Epoch transitions**:
  - reward payout for the completed epoch uses the completed epoch's
    `current_reward_settings`,
  - only after payout finishes does the module promote `next_reward_settings`
    into `current_reward_settings`,
  - explicit epoch state is stored so changing `blocks_in_epoch` does not break
    epoch continuity.
- **Safety checks**:
  - if total points in an epoch is zero, the epoch is advanced with no
    distribution,
  - if total voting power in a block is zero, no points are awarded.

## Default Genesis

`x/valrewards` exports a concrete `app_state.valrewards` object with both
config and runtime state.

Minimal shape:

```json
{
  "valrewards": {
    "params": {
      "whitelist": []
    },
    "current_reward_settings": {
      "blocks_in_epoch": 17280,
      "rewards_per_epoch": "45004521205479450000000",
      "rewarding_paused": false
    },
    "next_reward_settings": {
      "blocks_in_epoch": 17280,
      "rewards_per_epoch": "45004521205479450000000",
      "rewarding_paused": false
    },
    "epoch_state": {
      "current_epoch": 0,
      "blocks_into_current_epoch": 0
    },
    "epoch_to_pay": 0,
    "validator_points": [],
    "validator_outstanding_rewards": []
  }
}
```

Field meaning:

- `params.whitelist`: bech32 account addresses allowed to submit the runtime
  setter messages.
- `current_reward_settings`: active epoch settings.
- `next_reward_settings`: staged settings that activate at the next epoch
  rollover.
- `epoch_state.current_epoch`: epoch currently accumulating points.
- `epoch_state.blocks_into_current_epoch`: how many processed blocks have been
  counted inside the current epoch.
- `epoch_to_pay`: next epoch index expected by legacy payout bookkeeping.
- `validator_points`: persisted per-epoch validator point rows.
- `validator_outstanding_rewards`: persisted per-epoch validator outstanding
  reward rows.

Practical export/import notes:

- normal app export preserves params, active settings, staged settings, epoch
  state, validator points, and outstanding rewards,
- the rewards pool balance is not duplicated inside `app_state.valrewards`; it
  remains part of the module account balance in auth/bank state,
- zero-value outstanding reward rows may be omitted during export because
  missing and zero-valued entries are equivalent for this module.

## Validation Rules

- `blocks_in_epoch`
  - integer `int64`
  - minimum `20`
  - maximum `6500000`
- `rewards_per_epoch`
  - required non-empty decimal integer string
  - minimum `1000000000000000000`
  - maximum `25000000000000000000000000`
- `rewarding_paused`
  - boolean `true` or `false`
- `params.whitelist`
  - valid bech32 account addresses only
  - duplicate entries rejected
- `deposit amount`
  - must be a valid positive coin amount
- `validator_outstanding_rewards`
  - each entry must use `evmtypes.DefaultEVMDenom`
  - the sum of all outstanding rewards at genesis must not exceed the funded
    valrewards module account balance in that denom
- `epoch_to_pay`
  - cannot exceed `epoch_state.current_epoch`
- `validator_points` / `validator_outstanding_rewards`
  - entry epochs cannot exceed `epoch_state.current_epoch`

## Exposed Methods

### Cosmos SDK Msgs

- `MsgDepositRewardsPool(depositor, amount)`
- `MsgClaimRewards(validator_operator, epoch, requester)`
- `MsgSetBlocksInEpoch(signer, blocks_in_epoch)`
- `MsgSetRewardsPerEpoch(signer, rewards_per_epoch)`
- `MsgSetRewardingPaused(signer, rewarding_paused)`
- `MsgUpdateParams(authority, params)`

Authorization:

- deposit keeps its existing behavior,
- `MsgClaimRewards` is sponsor-callable: any valid requester may sign and
  submit the transaction, but rewards are still paid only to the target
  validator account,
- the three setter messages require the signer to be present in
  `params.whitelist`,
- `MsgUpdateParams` is authority-only, uses decoded address equality for final
  authorization, and is intended to be executed through governance.

### gRPC / Query

- `Query/RewardsPool`
- `Query/ValidatorOutstandingRewards`
- `Query/DelegationRewards`
- `Query/Params`

`Query/ValidatorOutstandingRewards` expects a canonical Bech32 validator
operator address. Hex `0x...` addresses are rejected.

`Query/Params` returns:

- `current_reward_settings`
- `next_reward_settings`
- `proposer_bonus_points`
- `whitelist`

All valrewards gRPC query handlers reject nil/empty requests with an invalid-request error.

### EVM Precompile

Address: `0x0000000000000000000000000000000000000714`

- `rewardsPool()`
- `validatorOutstandingRewards(epoch, validatorAddress)`
- `delegationRewards(delegatorAddress, epoch)`
- `depositValidatorRewardsPool(depositor, amount)`
- `claimRewards(validatorOperatorAddress, epoch)`

For the precompile `validatorOutstandingRewards` method:

- `validatorAddress` must be a canonical Bech32 validator operator address,
- hex `0x...` address input is rejected.

For the precompile `claimRewards` method:

- any EVM caller may submit the transaction,
- the target `validatorOperatorAddress` must resolve to a valid validator
  operator account,
- rewards are always paid to that target validator operator account, not to
  the caller.

## CLI

### Query

- `query valrewards rewards-pool`
- `query valrewards validator-outstanding-rewards [epoch] [validator-address]`
- `query valrewards delegation-rewards [delegator] [epoch]`
- `query valrewards params`

### Tx

- `tx valrewards deposit [amount]`
- `tx valrewards claim [validator-address] [epoch]`
- `tx valrewards set-blocks-in-epoch [blocks-in-epoch]`
- `tx valrewards set-rewards-per-epoch [rewards-per-epoch]`
- `tx valrewards set-rewarding-paused [true|false]`

Examples:

```bash
query valrewards params
tx valrewards set-blocks-in-epoch 25000 --from <whitelisted-account>
tx valrewards set-rewards-per-epoch 2000000000000000000 --from <whitelisted-account>
tx valrewards set-rewarding-paused true --from <whitelisted-account>
```

There is no standalone direct CLI whitelist mutation command for operators.
After chain start, whitelist changes must be submitted through governance using
`MsgUpdateParams`.

## Governance Whitelist Updates

Whitelist updates are governance-only after initialization.

The governance message is:

```json
{
  "@type": "/cosmos.evm.valrewards.v1.MsgUpdateParams",
  "authority": "<gov-module-address>",
  "params": {
    "whitelist": ["cosmos1..."]
  }
}
```

The signer must be the governance authority address, so this message is not a
normal operator tx flow. In practice it should be embedded in a governance
proposal.

## Test Scenarios

### Keeper and type tests

- params persistence and whitelist checks
- runtime setting validation
- staged-setting update auth for `blocks_in_epoch`, `rewards_per_epoch`, and
  `rewarding_paused`
- next-epoch activation semantics
- pause behavior
- multi-validator proportional reward splitting, including truncation behavior

### Precompile integration

- rewards pool query via precompile
- validator outstanding rewards via precompile
- delegation rewards via precompile
- deposit into rewards pool via precompile
- claim rewards via precompile and verify rewards are cleared

### CLI integration

- wait for a full epoch to accrue points
- query rewards pool
- query validator outstanding rewards
- query delegation rewards
- deposit into the rewards pool
- claim outstanding rewards
- update whitelist through governance
- stage `blocks_in_epoch`, `rewards_per_epoch`, and `rewarding_paused`
- verify `query valrewards params` shows current vs next settings correctly

### Governance whitelist integration

- verify a non-whitelisted account cannot submit a runtime setter tx
- approve a governance whitelist update and verify the newly whitelisted
  account can submit a runtime setter tx
- reject a governance whitelist update and verify the account remains blocked
- apply sequential governance whitelist updates and verify a later proposal can
  remove a previously whitelisted account
- submit a malformed governance whitelist proposal and verify it is rejected
- verify `query valrewards params` reflects whitelist changes and staged
  next-epoch values when proposals pass

### Export/import continuity

- populate `valrewards` runtime state
- run normal app export
- confirm exported state includes params, current settings, next settings, and
  epoch state
- start a fresh app from exported state
- verify rewards remain claimable after restart

## Test Commands

### Module tests

```bash
go test ./x/valrewards/... -count=1
```

### Precompile integration

```bash
go test -tags=test ./tests/integration/precompiles/valrewards -run TestValRewardsPrecompileIntegrationTestSuite -count=1
```

### CLI integration

```bash
go test -tags=test ./tests/integration/valrewards -run TestValRewardsCLIDemo -count=1
```

### Governance whitelist integration

```bash
go test -tags=test ./tests/integration/valrewards -run TestValRewardsWhitelistGovernance -count=1
```

### Export/import continuity

```bash
cd evmd && LD_LIBRARY_PATH=../rust/sp1verifier/target/release go test -run TestValRewards -count=1
```

Notes:

- `-tags=test` switches the module to test defaults so epochs complete quickly,
  with `BLOCKS_IN_EPOCH=20`,
- the `LD_LIBRARY_PATH` export is required in local environments where the SP1
  verifier shared library is not already on the shell library path.
