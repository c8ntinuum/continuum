# AddressTable Precompile Integration Scenarios

The AddressTable integration suite lives in `tests/integration/precompiles/addresstable/test_integration.go`
and is executed through the evmd wrapper test package:
`evmd/tests/integration/precompiles/addresstable/precompile_addresstable_test.go`.

Covered scenarios:

1. `register_and_address_exists`
- Register an address through AddressTable precompile `register(address)`.
- Verify the address is present via `addressExists(address)`.

2. `toggle_by_governance_and_restore`
- Register an address and verify `addressExists(address)`.
- Disable AddressTable precompile by governance (`x/vm` `MsgUpdateParams` removing `0x0000000000000000000000000000000000000710` from `active_static_precompiles`).
- Attempt `register(address)` and `addressExists(address)` while disabled; both must fail.
- Re-enable the precompile by governance (add address back to `active_static_precompiles`).
- Register a new address and verify `addressExists(address)` succeeds again.

Run commands (only this suite):

```bash
cd evmd
go test -tags=test ./tests/integration/precompiles/addresstable -run TestAddressTablePrecompileIntegrationTestSuite -v
```

Run individual scenario subtests:

```bash
cd evmd
go test -tags=test ./tests/integration/precompiles/addresstable -run 'TestAddressTablePrecompileIntegrationTestSuite/register_and_address_exists' -v
go test -tags=test ./tests/integration/precompiles/addresstable -run 'TestAddressTablePrecompileIntegrationTestSuite/toggle_by_governance_and_restore' -v
```
