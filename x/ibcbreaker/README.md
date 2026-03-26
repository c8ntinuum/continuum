# x/ibcbreaker

## Purpose

`x/ibcbreaker` provides an operator-controlled IBC breaker with:

- one global `ibc_available` flag
- a whitelist of operator accounts that can toggle the flag
- governance-controlled whitelist updates

The practical goal is to stop selected outbound IBC initiation paths, including
ICS20 transfer initiation (`MsgTransfer`), when the breaker is disabled.

## Known Non-Scope

`x/ibcbreaker` is not a full transfer freeze; internal non-IBC transfers (for example, bank `MsgSend`) remain available.

## State and Params

- `state.ibc_available`:
    - stored as a single byte (`1` for true, `0` for false)
    - if the key is missing, keeper defaults to `true`

- `params.whitelist`:
    - list of bech32 addresses allowed to submit `MsgUpdateIbcBreaker`
    - validated for address format and duplicates

Default genesis:

```json
{
  "ibcbreaker": {
    "params": {
      "whitelist": []
    },
    "state": {
      "ibc_available": true
    }
  }
}
```

## Authorization Model

- `MsgUpdateIbcBreaker`:
    - signer must be bech32
    - signer must be present in `params.whitelist`
    - otherwise returns unauthorized

- `MsgUpdateParams`:
    - authority must match module authority (governance module account)
    - `params` must be non-nil and valid
    - this is the only path to update whitelist

There is no direct tx command to edit whitelist; whitelist changes are expected
through governance messages (`MsgUpdateParams`).

## Enforcement Logic

Enforcement is implemented in two places:

- Cosmos ante decorator `IbcAvailableDecorator`
- transfer execution in `x/ibc/transfer` keeper (`Transfer` message path)

When `ibc_available=true`:

- ante checks pass
- transfer keeper allows `MsgTransfer` execution

When `ibc_available=false`:

- Cosmos ante scans tx messages
- restricted IBC message types are rejected with `ErrUnauthorized`
- `authz.MsgExec` contents are recursively scanned
- recursion depth >= 7 is rejected as unauthorized
- transfer keeper rejects `MsgTransfer` with `ErrUnauthorized` (`ibc unavailable`)

Restricted message types:

- `/ibc.core.client.v1.MsgCreateClient`
- `/ibc.core.connection.v1.MsgConnectionOpenInit`
- `/ibc.core.channel.v1.MsgChannelOpenInit`
- `/ibc.applications.transfer.v1.MsgTransfer`
- `/ibc.applications.interchain_accounts.controller.v1.MsgRegisterInterchainAccount`
- `/ibc.applications.interchain_accounts.controller.v1.MsgSendTx`
- `/ibc.core.client.v2.MsgRegisterCounterparty`
- `/ibc.core.client.v2.MsgUpdateClientConfig`
- `/ibc.core.channel.v2.MsgSendPacket`

## Scope and Non-Scope

- Cosmos tx route:
    - restricted list above is enforced by ante
    - `MsgTransfer` is also rejected in transfer keeper when breaker is disabled

- EVM extension tx route:
    - Cosmos IBC ante decorator is not applied
    - ICS20 precompile transfer still routes into transfer keeper, so `MsgTransfer` is blocked when breaker is disabled
- Not a full transfer freeze:
    - internal on-chain transfers remain possible
    - for example, bank `MsgSend` is not blocked by `x/ibcbreaker`

- Not all IBC message types are blocked:
    - only the restricted list above
    - non-restricted IBC messages can still pass this decorator

## Exposed Methods

Cosmos SDK messages:

- `MsgUpdateIbcBreaker(signer, ibc_available)`
- `MsgUpdateParams(authority, params)`

gRPC queries:

- `Query/IbcAvailable`
- `Query/Whitelist`

CLI:

- query:
    - `ctmd query ibcbreaker ibc-available`
    - `ctmd query ibcbreaker whitelist`
- tx:
    - `ctmd tx ibcbreaker update-ibcbreaker [true|false]`

## Test Coverage

The following tests cover currently implemented behavior.

### `TestIbcAvailableDecorator` (unit)

Location: `ante/cosmos/ibc_available_test.go`

Covers:

- breaker enabled allows restricted IBC messages
- breaker disabled rejects restricted IBC messages
- breaker disabled rejects ICS20 `MsgTransfer`
- breaker disabled allows non-restricted IBC messages (`MsgRecvPacket` case)
- breaker disabled allows non-IBC messages (`bank MsgSend` case)
- breaker disabled rejects restricted IBC inside `authz.MsgExec`
- breaker disabled rejects overly deep nested `authz.MsgExec`
- authz grant for non-IBC remains allowed

### `TestIbcBreakerCLIDemo` (integration, CLI flow)

Location: `ctmd/tests/integration/ibcbreaker/ibcbreaker_cli_test.go`

Covers:

- bootstraps chain with empty ibcbreaker whitelist and `ibc_available=true`
- governance proposal updates whitelist to include validator
- whitelisted validator toggles `ibc_available` false then true
- non-whitelisted account cannot toggle breaker
- governance proposal adds second account to whitelist
- newly whitelisted second account can toggle breaker
- query checks validate whitelist and flag values after each step

### `TestIbcBreakerWhitelistGovernance` (integration, keeper/msg-server flow)

Location: `ctmd/tests/integration/ibcbreaker/ibcbreaker_governance_test.go`
Helper: `tests/integration/ibcbreaker/governance_helper.go`

Covers:

- genesis whitelist account can toggle breaker
- non-whitelisted account cannot toggle breaker
- non-governance authority cannot update whitelist params directly
- governance `MsgUpdateParams` adds second account to whitelist
- second account can toggle breaker after proposal passes

### `TestTransferBlockedWhenIbcUnavailable` (integration, transfer keeper)

Location: `tests/integration/x/ibc/test_msg_server.go`
Run via: `ctmd/tests/integration/ibc_test.go`

Covers:

- direct `MsgTransfer` execution path rejects when breaker is disabled
- validates transfer keeper level enforcement (`ibc unavailable`)

### `TransferTestSuite/TestHandleMsgTransferBlockedWhenIbcUnavailable` (integration, native Cosmos tx route)

Location: `ctmd/tests/ibc/transfer_test.go`

Covers:

- Cosmos `MsgTransfer` transaction fails when breaker is disabled

- validates native tx route rejection (`ibc unavailable`)

### `ICS20TransferTestSuite/TestHandleMsgTransferBlockedWhenIbcUnavailable` (integration, EVM precompile route)

Location: `ctmd/tests/ibc/ics20_precompile_transfer_test.go`

Covers:

- ICS20 precompile transfer on EVM route fails when breaker is disabled
- confirms EVM-originated transfer path is blocked with `ibc unavailable`

## Requirement Coverage Matrix

| Requirement | Enforcement Point(s) | Tests |
| --- | --- | --- |
| Whitelist set at genesis; whitelist changes by governance only | `InitGenesis`, `MsgUpdateParams` authority check | `TestIbcBreakerCLIDemo`, `TestIbcBreakerWhitelistGovernance` |
| CLI only exposes status/whitelist queries and breaker toggle tx | ibcbreaker CLI query/tx commands | `TestIbcBreakerCLIDemo` |
| Breaker engaged blocks IBC money-out on native and EVM transfer routes | Cosmos ante restricted IBC list + transfer keeper `MsgTransfer` guard | `TestIbcAvailableDecorator`, `TransferTestSuite/TestHandleMsgTransferBlockedWhenIbcUnavailable`, `ICS20TransferTestSuite/TestHandleMsgTransferBlockedWhenIbcUnavailable`, `TestTransferBlockedWhenIbcUnavailable` |
| Only whitelisted addresses can toggle breaker | `MsgUpdateIbcBreaker` whitelist check | `TestIbcBreakerCLIDemo`, `TestIbcBreakerWhitelistGovernance` |

## Test Summary

Run unit enforcement test:

```bash
go test ./ante/cosmos -run TestIbcAvailableDecorator -count=1
```

Run direct transfer keeper blocking test:

```bash
go test -tags=test ./tests/integration -run TestIBCKeeperTestSuite/TestTransferBlockedWhenIbcUnavailable -count=1
```

Run native Cosmos tx route blocking test:

```bash
go test -tags=test ./tests/ibc -run TestTransferTestSuite/TestHandleMsgTransferBlockedWhenIbcUnavailable -count=1
```

Run EVM ICS20 precompile blocking test:

```bash
go test -tags=test ./tests/ibc -run TestICS20TransferTestSuite/TestHandleMsgTransferBlockedWhenIbcUnavailable -count=1
```

Run CLI integration demo:

```bash
go test -tags=test ./tests/integration/ibcbreaker -run TestIbcBreakerCLIDemo -count=1
```

Run governance integration test:

```bash
go test -tags=test ./tests/integration/ibcbreaker -run TestIbcBreakerWhitelistGovernance -count=1
```
