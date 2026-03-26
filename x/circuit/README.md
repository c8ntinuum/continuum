# x/circuit

## Purpose

`x/circuit` is a simple circuit breaker for the chain. It exposes a global
`system_available` flag and a whitelist of accounts authorized to toggle it.
Whitelist management is governance-controlled.

## Core Logic

- **System Availability Flag**: a single boolean stored in module state.
- **Whitelist Enforcement**: only whitelisted accounts can submit
  `MsgUpdateCircuit` to flip the flag.
- **Governance-Only Params**: whitelist updates happen via
  `MsgUpdateParams` and require the governance authority address.

## Enforcement Scope

- The circuit gate is enforced in ante for both transaction routes:
    - Native Cosmos SDK transaction route.
    - EVM extension transaction route (`ExtensionOptionsEthereumTx`).
- `eth_sendRawTransaction` submissions are converted into `MsgEthereumTx` and
  pass through ante checks, so they are also gated by `system_available`.
- When `system_available=false`, only transactions containing
  `MsgUpdateCircuit` messages are allowed through the circuit gate.
- Any transaction containing non-`MsgUpdateCircuit` messages is rejected with
  `ErrUnauthorized` ("system unavailable").

## Default Genesis

```json
{
  "circuit": {
    "params": {
      "whitelist": []
    },
    "state": {
      "system_available": true
    }
  }
}
```

- `params.whitelist`: bech32 account addresses allowed to submit `MsgUpdateCircuit`.
  Default is empty (`[]`), so only governance can change authorization via params updates.
- `state.system_available`: global circuit breaker flag for system availability.
  Default is `true`, so the system starts in available mode.

## Exposed Methods

### Cosmos SDK Msgs

- `MsgUpdateCircuit(signer, system_available)`  
  Toggle the global circuit flag. Rejected if the signer is not whitelisted.
- `MsgUpdateParams(authority, params)`  
  Update module params (currently the whitelist). Authority must be the
  governance module address.

### gRPC Query

- `Query/SystemAvailable`  
  Returns current `system_available` value.
- `Query/Whitelist`  
  Returns the whitelist array.

### CLI

- Query
    - `query circuit system-available`
    - `query circuit whitelist`
- Tx
    - `tx circuit update-circuit [true|false]`

## Route Coverage Matrix

| Route | Current Coverage | Test(s) |
| --- | --- | --- |
| Native / CLI circuit tx (`tx circuit update-circuit`) | Yes (integration) | `TestCircuitCLIDemo` |
| Governance whitelist update (`MsgUpdateParams`) | Yes (integration) | `TestCircuitCLIDemo`, `TestCircuitWhitelistGovernance` |
| Ante gate logic for `system_available` | Yes (unit) | `TestCircuitAvailableDecorator` |
| Control-path tx allow while unavailable (`MsgUpdateCircuit`) | Yes (integration) | `TestCircuitCLIDemo` |
| EVM message type gate behavior in decorator | Yes (unit) | `TestCircuitAvailableDecorator` |
| EVM tx end-to-end route (`ExtensionOptionsEthereumTx`) | Yes (integration) | `TestCircuitCLIDemo` |
| EVM tx via JSON-RPC (`eth_sendRawTransaction`) | Yes (integration) | `TestCircuitCLIDemo` |
| Native non-circuit tx end-to-end block when disabled | Yes (integration) | `TestCircuitCLIDemo` |
| `system_available` query correctness | Yes (integration) | `TestCircuitCLIDemo` |
| `whitelist` query correctness | Yes (integration) | `TestCircuitCLIDemo` |

All currently implemented `x/circuit` behaviors are covered by the test set above.

## Test Commands

### CLI demo

```bash
go test -tags=test ./tests/integration/circuit -run TestCircuitCLIDemo -count=1
```

**Params**

- `-tags=test`: enables integration test build tags and test-only config tweaks.
- `./tests/integration/circuit`: runs only circuit integration tests (from the `ctmd` module).
- `-run TestCircuitCLIDemo`: runs only the CLI demo test.
- `-count=1`: disables test caching for a fresh run.

### Governance helper

```bash
go test -tags=test ./tests/integration/circuit -run TestCircuitWhitelistGovernance -count=1
```

**Params**

- `-tags=test`: enables integration test build tags and test-only config tweaks.
- `./tests/integration/circuit`: runs only circuit integration tests (from the `ctmd` module).
- `-run TestCircuitWhitelistGovernance`: runs only the governance helper test.
- `-count=1`: disables test caching for a fresh run.

### Ante decorator unit

```bash
go test ./ante/cosmos -run TestCircuitAvailableDecorator -count=1
```

**Params**

- `./ante/cosmos`: runs ante Cosmos decorators package tests.
- `-run TestCircuitAvailableDecorator`: runs only the circuit availability decorator unit test.
- `-count=1`: disables test caching for a fresh run.

## Test Summary

Copy/paste commands:

```bash
go test -tags=test ./tests/integration/circuit -run TestCircuitCLIDemo -count=1
go test -tags=test ./tests/integration/circuit -run TestCircuitWhitelistGovernance -count=1
go test ./ante/cosmos -run TestCircuitAvailableDecorator -count=1
```

Run all `x/circuit` focused tests:

```bash
go test -tags=test ./tests/integration/circuit -count=1
go test ./ante/cosmos -count=1
```
