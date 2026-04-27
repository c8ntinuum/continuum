###############################################################################
# Cleaning
###############################################################################

GO ?= go

.PHONY: clean
clean:
	@echo "🧹  Cleaning build artifacts..."
	@rm -rf "$(BUILDDIR)" "$(RELEASEDIR)"
	@if command -v "$(RUST_SP1_CARGO_BIN)" >/dev/null 2>&1; then \
		cd "$(RUST_SP1_DIR)" && $(RUST_SP1_CARGO_BIN) clean; \
	else \
		echo "ℹ️  cargo not found; skipping Rust clean"; \
	fi
	@if command -v "$(GO)" >/dev/null 2>&1; then \
		cd "$(CTM_DIR)" && $(GO) clean ./...; \
	else \
		echo "ℹ️  go not found; skipping Go clean"; \
	fi
