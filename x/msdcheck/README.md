# x/msdcheck

This module enforces a minimum self-delegation floor for validator creation and for proposed `MinSelfDelegation` updates across **all** transaction entry points (Cosmos SDK txs, API, EVM precompile).

The enforced minimum is **888888000000000000000000**.

## Scope And Semantics

- The rule applies to the **validator operator's self-delegation only**.
- It does **not** depend on delegations from other users.
- A validator can only set or raise `MinSelfDelegation` via create/edit.
- `EditValidator` is not blocked solely because the validator's current tokens fell below the threshold after creation; the restriction applies to the proposed `MinSelfDelegation` value, not to unrelated edit operations.
- If the operator later reduces their self-delegation below the minimum (e.g. via undelegation), the **staking module's native behavior** will jail the validator (remove from active set). This module does not block undelegate or redelegate.

## Default Genesis

`x/msdcheck` does not have an independent module genesis section in `app_state`.
It wraps/extends staking message handling and migrations.

Practical implication:

- there is no `app_state.msdcheck` object to configure.
- staking genesis (`app_state.staking`) still applies as usual.
- the minimum self-delegation check (`888888000000000000000000`) is enforced by module logic, not by msdcheck-specific genesis params.

## Test Scenarios

The test suite covers:

1. `MsgCreateValidator` below minimum — **reject**
2. `MsgCreateValidator` at exact minimum — **accept**
3. `MsgCreateValidator` above minimum — **accept**
4. `MsgEditValidator` below current stake threshold but with no invalid `MinSelfDelegation` change — **accept**
5. `MsgEditValidator` proposed `MinSelfDelegation` below minimum — **reject**
6. EVM staking precompile `createValidator` below minimum — **reject**
7. EVM staking precompile `createValidator` at exact minimum — **accept**
8. EVM staking precompile `createValidator` above minimum — **accept**

## Test Commands

Run Cosmos-side MSD tests:

From repo root:

```bash
go test -tags=test ./tests/integration/staking -run MSDCheck -count=1
```

Explanation:

- `-tags=test`: enables integration test build flags required by the network harness.
- `./tests/integration/staking`: runs only the Cosmos-side MSD integration tests.
- `-run MSDCheck`: runs only the MSD test suite entrypoint.
- `-count=1`: disables test caching for a fresh run.

Run EVM precompile MSD tests:

From repo root:

```bash
go test -tags=test ./tests/integration/precompiles/staking \
  -run TestStakingPrecompileIntegrationTestSuite \
  -ginkgo.label-filter=msd \
  -count=1
```

Explanation:

- `-tags=test`: enables integration test build flags required by the network harness.
- `./tests/integration/precompiles/staking`: runs staking precompile integration tests.
- `-run TestStakingPrecompileIntegrationTestSuite`: only runs the staking precompile suite entrypoint.
- `-ginkgo.label-filter=msd`: runs only the specs labeled `msd`.
- `-count=1`: disables test caching for a fresh run.
