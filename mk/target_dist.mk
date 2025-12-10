###############################################################################
# dist: self-contained bundle in $(DISTDIR)
###############################################################################

GO ?= go
GOFLAGS ?=
UNAME_S ?= $(shell uname -s)

.PHONY: dist
dist: rust-sp1
	@echo "📦  Assembling dist layout in $(DISTDIR)"
	@rm -rf "$(DISTDIR)"
	@mkdir -p "$(DIST_BINDIR)" "$(DIST_LIBDIR)"

	@echo "🏗️  Building $(CTM_BINARY) -> $(DIST_BINDIR)/$(CTM_BINARY)"
	@cd $(EVMD_DIR) && CGO_ENABLED="1" \
		$(GO) build $(GOFLAGS) $(BUILD_FLAGS) -o $(DIST_BINDIR)/$(CTM_BINARY) $(EVMD_MAIN_PKG)

	@echo "📦  Copying Rust FFI lib -> $(DIST_LIBDIR)/$(RUST_SP1_LIB_BASENAME)"
	@cp "$(RUST_SP1_LIB_BUILD)" "$(DIST_LIBDIR)/$(RUST_SP1_LIB_BASENAME)"

	@echo "🔧  Bundling + fixup"
	@$(BUNDLE_AND_FIXUP) \
		"$(DIST_BINDIR)/$(CTM_BINARY)" "$(DIST_LIBDIR)" \
		"$(DIST_DARWIN_RPATH)" '$(DIST_LINUX_RPATH)'

	@echo "✅  Dist bundle ready: $(DISTDIR)"
