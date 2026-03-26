###############################################################################
# vendor: download and lock all Go + Rust dependencies locally
###############################################################################

GO ?= go
CARGO ?= cargo
RUST_SP1_DIR ?= $(ROOT)/rust/sp1verifier

.PHONY: vendor vendor-go vendor-rust vendor-clean

vendor: vendor-go vendor-rust

vendor-go:
	@echo "📦  Vendoring Go dependencies in $(CTM_DIR) ..."
	@cd "$(CTM_DIR)" && $(GO) mod vendor

vendor-rust:
	@echo "📦  Vendoring Rust dependencies in $(RUST_SP1_DIR) ..."
	@cd "$(RUST_SP1_DIR)" && $(CARGO) vendor
	@mkdir -p "$(RUST_SP1_DIR)/.cargo"
	@printf '[source.crates-io]\nreplace-with = "vendored-sources"\n\n[source.vendored-sources]\ndirectory = "vendor"\n' \
		> "$(RUST_SP1_DIR)/.cargo/config.toml"
	@echo "  Wrote $(RUST_SP1_DIR)/.cargo/config.toml"

vendor-clean:
	@echo "🧹  Removing vendored dependencies ..."
	@rm -rf "$(CTM_DIR)/vendor"
	@rm -rf "$(RUST_SP1_DIR)/vendor"
	@rm -f "$(RUST_SP1_DIR)/.cargo/config.toml"
