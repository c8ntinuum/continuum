# x/circuit

## Purpose

`x/circuit` is a simple circuit breaker for the chain. It exposes a global
`system_available` flag and a whitelist of accounts authorized to toggle it.
Whitelist management is governance-controlled.

## Core Logic

- **System Availability Flag**: a single boolean stored in module state.
- **Whitelist Enforcement**: only whitelisted accounts can submit
  `MsgUpdateCircuit` to flip the flag.
- **No-Op Toggle Guard**: `MsgUpdateCircuit` returns success without rewriting
  state when the requested `system_available` value already matches current state.
- **Recovery-Safe Empty Whitelist Rule**: an empty whitelist is allowed while
  `system_available=true`, but it is rejected when `system_available=false`
  so the chain cannot be left without any operator able to re-enable it.
- **Defensive Query Validation**: gRPC query handlers reject nil request
  objects explicitly instead of depending on upstream transport behavior.
- **Governance-Only Params**: whitelist updates happen via
  `MsgUpdateParams` and require the governance authority address.
- **Canonical Authority Matching**: keeper-side governance authorization for
  `MsgUpdateParams` parses the provided bech32 authority and compares address
  identity instead of relying on raw string equality.
- **Defensive Msg Validation**: `MsgUpdateParams.ValidateBasic()` rejects
  nil `params` before handler execution so malformed transactions return an
  invalid-request error instead of panicking.

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
  Default is empty (`[]`), which is valid while the system is online. This lets
  governance define operators later. However, an empty whitelist is invalid when
  `state.system_available=false`, because at least one whitelisted operator must
  remain able to submit `MsgUpdateCircuit` and restore availability.
- `state.system_available`: global circuit breaker flag for system availability.
  Default is `true`, so the system starts in available mode.

## Exposed Methods

### Cosmos SDK Msgs

- `MsgUpdateCircuit(signer, system_available)`  
  Toggle the global circuit flag. Rejected if the signer is not whitelisted.
  If the requested value already matches current state, the call succeeds as a no-op.
- `MsgUpdateParams(authority, params)`  
  Update module params (currently the whitelist). Authority must be the
  governance module address. An empty whitelist is accepted only while the
  system is currently available.

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
| `MsgUpdateCircuit` no-op state write guard | Yes (unit) | `TestUpdateCircuitSkipsNoOpStateWrite` |
| Keeper nil-request and whitelist rejection edges | Yes (unit) | `TestUpdateCircuitRejectsNonWhitelistedSigner`, `TestUpdateCircuitRejectsNilRequest`, `TestUpdateParamsRejectsNilRequest` |
| Query nil-request guards | Yes (unit) | `TestSystemAvailableRejectsNilRequest`, `TestWhitelistRejectsNilRequest` |
| Governance whitelist update (`MsgUpdateParams`) | Yes (integration) | `TestCircuitCLIDemo`, `TestCircuitWhitelistGovernance` |
| Keeper authority canonicalization for `MsgUpdateParams` | Yes (unit + integration) | `TestUpdateParamsAuthorityCanonicalization`, `TestCircuitWhitelistGovernance` |
| `MsgUpdateParams.ValidateBasic` nil params guard | Yes (unit) | `TestMsgsTestSuite/TestMsgUpdateParamsValidateBasic` |
| Empty whitelist rejected while offline | Yes (unit + genesis validation) | `TestUpdateParamsRejectsEmptyWhitelistWhileSystemUnavailable`, `TestGenesisStateValidate` |
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

### Msg validation unit

```bash
go test ./x/circuit/types -run TestMsgsTestSuite/TestMsgUpdateParamsValidateBasic -count=1
```

**Params**

- `./x/circuit/types`: runs the circuit message type unit tests.
- `-run TestMsgsTestSuite/TestMsgUpdateParamsValidateBasic`: runs the `MsgUpdateParams` basic validation coverage, including the nil `params` guard.
- `-count=1`: disables test caching for a fresh run.

### Keeper auth unit

```bash
go test ./x/circuit/keeper -run TestUpdateParamsAuthorityCanonicalization -count=1
```

**Params**

- `./x/circuit/keeper`: runs the circuit keeper unit tests.
- `-run TestUpdateParamsAuthorityCanonicalization`: verifies that keeper-side authority checks canonicalize bech32 input before authorization comparison.
- `-count=1`: disables test caching for a fresh run.

### Keeper toggle no-op unit

```bash
go test ./x/circuit/keeper -run TestUpdateCircuitSkipsNoOpStateWrite -count=1
```

**Params**

- `./x/circuit/keeper`: runs the circuit keeper unit tests.
- `-run TestUpdateCircuitSkipsNoOpStateWrite`: verifies that `MsgUpdateCircuit` returns success without rewriting state when the requested value already matches current state.
- `-count=1`: disables test caching for a fresh run.

### Keeper recovery guard unit

```bash
go test ./x/circuit/keeper -run TestUpdateParamsRejectsEmptyWhitelistWhileSystemUnavailable -count=1
```

**Params**

- `./x/circuit/keeper`: runs the circuit keeper unit tests.
- `-run TestUpdateParamsRejectsEmptyWhitelistWhileSystemUnavailable`: verifies that an empty whitelist is rejected only while the system is unavailable, and remains valid while the system is online.
- `-count=1`: disables test caching for a fresh run.

### Keeper edge-case units

```bash
go test ./x/circuit/keeper -run 'TestUpdateCircuitRejectsNonWhitelistedSigner|TestUpdateCircuitRejectsNilRequest|TestUpdateParamsRejectsNilRequest' -count=1
```

**Params**

- `./x/circuit/keeper`: runs the circuit keeper unit tests.
- `-run 'TestUpdateCircuitRejectsNonWhitelistedSigner|TestUpdateCircuitRejectsNilRequest|TestUpdateParamsRejectsNilRequest'`: verifies remaining keeper edge cases for unauthorized signers and nil requests.
- `-count=1`: disables test caching for a fresh run.

### Query guard units

```bash
go test ./x/circuit/keeper -run 'TestSystemAvailableRejectsNilRequest|TestWhitelistRejectsNilRequest' -count=1
```

**Params**

- `./x/circuit/keeper`: runs the circuit keeper and query unit tests.
- `-run 'TestSystemAvailableRejectsNilRequest|TestWhitelistRejectsNilRequest'`: verifies that both circuit query handlers reject nil request objects explicitly.
- `-count=1`: disables test caching for a fresh run.

### Genesis validation unit

```bash
go test ./x/circuit/types -run TestGenesisStateValidate -count=1
```

**Params**

- `./x/circuit/types`: runs the circuit type and genesis unit tests.
- `-run TestGenesisStateValidate`: verifies that offline genesis requires at least one whitelisted operator, while online genesis may still use an empty whitelist.
- `-count=1`: disables test caching for a fresh run.

## Test Summary

Copy/paste commands:

```bash
go test -tags=test ./tests/integration/circuit -run TestCircuitCLIDemo -count=1
go test -tags=test ./tests/integration/circuit -run TestCircuitWhitelistGovernance -count=1
go test ./ante/cosmos -run TestCircuitAvailableDecorator -count=1
go test ./x/circuit/types -run TestMsgsTestSuite/TestMsgUpdateParamsValidateBasic -count=1
go test ./x/circuit/types -run TestGenesisStateValidate -count=1
go test ./x/circuit/keeper -run TestUpdateParamsAuthorityCanonicalization -count=1
go test ./x/circuit/keeper -run TestUpdateCircuitSkipsNoOpStateWrite -count=1
go test ./x/circuit/keeper -run TestUpdateParamsRejectsEmptyWhitelistWhileSystemUnavailable -count=1
go test ./x/circuit/keeper -run 'TestUpdateCircuitRejectsNonWhitelistedSigner|TestUpdateCircuitRejectsNilRequest|TestUpdateParamsRejectsNilRequest' -count=1
go test ./x/circuit/keeper -run 'TestSystemAvailableRejectsNilRequest|TestWhitelistRejectsNilRequest' -count=1
```

Run all `x/circuit` focused tests:

```bash
go test -tags=test ./tests/integration/circuit -count=1
go test ./ante/cosmos -count=1
go test ./x/circuit/types -count=1
go test ./x/circuit/keeper -count=1
```
