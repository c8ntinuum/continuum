###############################################################################
# install: user-safe install
#   macOS:  $(BINDIR)/ctmd + $(BINDIR)/Frameworks/*.dylib
#   Linux:  $(BINDIR)/ctmd + $(BINDIR)/lib/*.so*
###############################################################################

GO ?= go
GOFLAGS ?=
UNAME_S ?= $(shell uname -s)

.PHONY: install install-system

install: rust-sp1
	@echo "🚚  Installing $(CTM_BINARY) to $(BINDIR) ..."
	@mkdir -p "$(BINDIR)"
	@cd $(EVMD_DIR) && CGO_ENABLED="1" \
		$(GO) build $(GOFLAGS) $(BUILD_FLAGS) -o $(BINDIR)/$(CTM_BINARY) $(EVMD_MAIN_PKG)

	@set -e; \
	if [ "$(UNAME_S)" = "Darwin" ]; then \
	  LIBDIR="$(BINDIR)/Frameworks"; \
	else \
	  LIBDIR="$(BINDIR)/lib"; \
	fi; \
	echo "📦  Installing bundled libs to $$LIBDIR"; \
	mkdir -p "$$LIBDIR"; \
	cp "$(RUST_SP1_LIB_BUILD)" "$$LIBDIR/$(RUST_SP1_LIB_BASENAME)"; \
	"$(BUNDLE_AND_FIXUP)" \
		"$(BINDIR)/$(CTM_BINARY)" "$$LIBDIR" \
		"$(INSTALL_DARWIN_RPATH)" '$(INSTALL_LINUX_RPATH)'

	@echo "✅  Installed: $(BINDIR)/$(CTM_BINARY)"

install-system: rust-sp1
	@echo "🚚  Installing $(CTM_BINARY) to $(BINDIR) ..."
	@mkdir -p "$(BINDIR)"
	@cd $(EVMD_DIR) && CGO_ENABLED="1" \
		$(GO) build $(GOFLAGS) $(BUILD_FLAGS) -o $(BINDIR)/$(CTM_BINARY) $(EVMD_MAIN_PKG)
	@if [ "$(UNAME_S)" = "Linux" ]; then \
	  echo "📦  Installing $(RUST_SP1_LIB_BASENAME) to $(LINUX_LIBDIR) (sudo may be required)"; \
	  install -d "$(LINUX_LIBDIR)"; \
	  install -m 0755 "$(RUST_SP1_LIB_BUILD)" "$(LINUX_LIBDIR)/$(RUST_SP1_LIB_BASENAME)"; \
	  command -v ldconfig >/dev/null 2>&1 && ldconfig || true; \
	else \
	  echo "install-system is intended for Linux; use 'make install' on macOS."; \
	fi
