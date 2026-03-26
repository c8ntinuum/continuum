###############################################################################
# reproducibility-env: print a JSON toolchain fingerprint
###############################################################################

GO ?= go
CARGO ?= cargo

.PHONY: reproducibility-env

reproducibility-env:
	@printf '{\n'
	@printf '  "go_version": "%s",\n' "$$($(GO) version 2>/dev/null || echo unknown)"
	@printf '  "rust_version": "%s",\n' "$$(rustc --version 2>/dev/null || echo unknown)"
	@printf '  "cargo_version": "%s",\n' "$$($(CARGO) --version 2>/dev/null || echo unknown)"
	@printf '  "gcc_version": "%s",\n' "$$(gcc --version 2>/dev/null | head -1 || echo unknown)"
	@printf '  "rocksdb_version": "%s",\n' "$(ROCKSDB_VERSION)"
	@printf '  "os": "%s",\n' "$$(uname -s)"
	@printf '  "arch": "%s",\n' "$$(uname -m)"
	@printf '  "os_version": "%s",\n' "$$(uname -r)"
	@printf '  "timestamp": "%s"\n' "$$(date -u +%Y-%m-%dT%H:%M:%SZ)"
	@printf '}\n'
