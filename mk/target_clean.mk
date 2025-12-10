###############################################################################
# Cleaning
###############################################################################

GO ?= go

.PHONY: clean
clean:
	@echo "🧹  Cleaning build artifacts..."
	@rm -rf "$(BUILDDIR)"
	@cd "$(RUST_SP1_DIR)" && $(CARGO) clean
	@cd "$(EVMD_DIR)" && $(GO) clean ./...
