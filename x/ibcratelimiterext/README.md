# x/ibcratelimiterext

## Purpose

`x/ibcratelimiterext` adds an independent, governance-managed operator whitelist for IBC rate-limiter operations.

Behavior:

- packet enforcement remains the same as upstream `ratelimiting` middleware,
- whitelist is stored in `x/ibcratelimiterext` state (not `x/circuit`),
- governance updates whitelist via `MsgUpdateParams`,
- whitelisted operators can call the same rate-limiter mutation APIs.

## What "operate the component" means

An address can operate the rate limiter if it can execute:

- `MsgAddRateLimit`
- `MsgUpdateRateLimit`
- `MsgRemoveRateLimit`
- `MsgResetRateLimit`

Authorization rule:

1. signer is ratelimiting authority (gov authority), OR
2. signer is in `ibcratelimiterext` whitelist.

## Architecture

## 1. Independent whitelist module

`x/ibcratelimiterext` (module name `ibcratelimiterext`) owns:

- `Params.whitelist` in its own store/genesis,
- `MsgUpdateParams` (authority-only),
- `QueryWhitelist`.

## 2. Ratelimiter service override

`ibcratelimiterext.NewRateLimitingAppModule(...)` keeps ratelimiting data plane + queries/genesis, but overrides `ratelimit.v1.Msg` authorization through `x/ibcratelimiterext/keeper/msg_server.go`.

## 3. Middleware stack

Unchanged packet path:

- v1: `transfer -> erc20 -> callbacks -> ratelimiting -> channel`
- v2: `transferv2 -> erc20v2 -> ratelimitingv2`

## CLI

## Query CLI

Your component exposes its own query root and includes rate-limiter read commands too.

Root:

- `ctmd query ibcratelimiterext ...`

### Whitelist query (module-owned)

```bash
ctmd query ibcratelimiterext whitelist --node <rpc> --chain-id <chain-id>
```

### Included rate-limiter read commands (same as imported module)

```bash
ctmd query ibcratelimiterext list-rate-limits --node <rpc> --chain-id <chain-id>
ctmd query ibcratelimiterext rate-limit channel-0 --denom ibc/ABCDEF123 --node <rpc> --chain-id <chain-id>
ctmd query ibcratelimiterext rate-limit channel-0 --node <rpc> --chain-id <chain-id>
ctmd query ibcratelimiterext rate-limits-by-chain osmosis-1 --node <rpc> --chain-id <chain-id>
ctmd query ibcratelimiterext list-blacklisted-denoms --node <rpc> --chain-id <chain-id>
ctmd query ibcratelimiterext list-whitelisted-addresses --node <rpc> --chain-id <chain-id>
```

## Tx CLI

Root:

- `ctmd tx ibcratelimiterext ...`

### Governance whitelist update

`MsgUpdateParams` is authority-only:

```bash
ctmd tx ibcratelimiterext update-params \
  --whitelist "cosmos1...,cosmos1..." \
  --from <gov-authority-wallet> \
  --chain-id <chain-id> \
  --node <rpc> \
  --gas auto --gas-adjustment 1.4 \
  --fees 5000stake
```

### Rate-limit write commands (same operations as imported ratelimiting)

These are direct operator commands. Signer must be either gov authority or in `ibcratelimiterext` whitelist.

```bash
# Add limit
ctmd tx ibcratelimiterext add-rate-limit channel-0 ibc/ABCDEF123 \
  --max-percent-send 10 \
  --max-percent-recv 20 \
  --duration-hours 24 \
  --from <operator-wallet> \
  --chain-id <chain-id> \
  --node <rpc> \
  --gas auto --gas-adjustment 1.4 \
  --fees 5000stake

# Update limit
ctmd tx ibcratelimiterext update-rate-limit channel-0 ibc/ABCDEF123 \
  --max-percent-send 15 \
  --max-percent-recv 25 \
  --duration-hours 24 \
  --from <operator-wallet> \
  --chain-id <chain-id> \
  --node <rpc> \
  --gas auto --gas-adjustment 1.4 \
  --fees 5000stake

# Remove limit
ctmd tx ibcratelimiterext remove-rate-limit channel-0 ibc/ABCDEF123 \
  --from <operator-wallet> \
  --chain-id <chain-id> \
  --node <rpc> \
  --gas auto --gas-adjustment 1.4 \
  --fees 5000stake

# Reset limit flow window
ctmd tx ibcratelimiterext reset-rate-limit channel-0 ibc/ABCDEF123 \
  --from <operator-wallet> \
  --chain-id <chain-id> \
  --node <rpc> \
  --gas auto --gas-adjustment 1.4 \
  --fees 5000stake
```

## Default Genesis

```json
{
  "ibcratelimiterext": {
    "params": {
      "whitelist": []
    }
  }
}
```

- `params.whitelist`: bech32 account addresses allowed to execute ratelimiter
  write operations (`add/update/remove/reset-rate-limit`) through this extension.
  Default is empty (`[]`), so only ratelimiter authority (governance authority)
  can operate those commands until whitelist entries are added.

Note: the upstream `ratelimiting` module has its own separate genesis in
`app_state.ratelimiting` (including `hour_epoch`). `ibcratelimiterext` does not store
or initialize those ratelimiting fields.

### Minimal Combined Genesis Example (`ratelimiting` + `ibcratelimiterext`)

```json
{
  "app_state": {
    "ratelimiting": {
      "rate_limits": [],
      "whitelisted_address_pairs": [],
      "blacklisted_denoms": [],
      "pending_send_packet_sequence_numbers": [],
      "hour_epoch": {
        "epoch_number": "0",
        "duration": "3600s",
        "epoch_start_time": "0001-01-01T00:00:00Z",
        "epoch_start_height": "0"
      }
    },
    "ibcratelimiterext": {
      "params": {
        "whitelist": []
      }
    }
  }
}
```

- `app_state.ratelimiting`: required base module state; without it, ratelimiting
  `BeginBlocker` logs `hour epoch not found in store`.
- `app_state.ibcratelimiterext`: extension whitelist state for operator auth.

## Files Added/Changed

Core module:

- `x/ibcratelimiterext/module.go`
- `x/ibcratelimiterext/genesis.go`
- `x/ibcratelimiterext/ratelimiting_module.go`
- `x/ibcratelimiterext/keeper/keeper.go`
- `x/ibcratelimiterext/keeper/msg_server_params.go`
- `x/ibcratelimiterext/keeper/grpc_query.go`
- `x/ibcratelimiterext/keeper/msg_server.go`
- `x/ibcratelimiterext/client/cli/query.go`
- `x/ibcratelimiterext/client/cli/tx.go`

Types/proto outputs:

- `x/ibcratelimiterext/types/*.go`
- `proto/cosmos/evm/ibcratelimiterext/v1/*.proto`

App wiring:

- `ctmd/app.go`

## Tests

## Test files and purpose

1. `x/ibcratelimiterext/keeper/keeper_test.go`

- validates params persistence and `IsWhitelisted` behavior.

2. `x/ibcratelimiterext/keeper/msg_server_params_test.go`

- validates `MsgUpdateParams` authority enforcement.

3. `x/ibcratelimiterext/keeper/msg_server_test.go`

- validates ratelimiter operation auth with extension whitelist:
- authority allowed,
- whitelisted operator allowed,
- unauthorized denied,
- remove-not-found behavior,
- reset allowed for whitelisted operator.

## Commands to run

Targeted extension tests:

```bash
go test ./x/ibcratelimiterext/... -v
```

Only whitelist params tests:

```bash
go test ./x/ibcratelimiterext/keeper -run "TestParamsRoundTrip|TestIsWhitelisted TestUpdateParams_AuthorityOnly" -v
```

Only ratelimiter auth tests:

```bash
go test ./x/ibcratelimiterext/keeper -run "TestAddRateLimit|TestRemoveRateLimit|TestResetRateLimit" -v
```
