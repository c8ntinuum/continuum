###############################################################################
# dist: self-contained bundle in $(DISTDIR)
###############################################################################

GO ?= go
GOFLAGS ?=
UNAME_S ?= $(shell uname -s)
UNAME_M ?= $(shell uname -m)


ifeq ($(UNAME_S),Darwin)
  DIST_PLATFORM_OS := darwin
else ifeq ($(UNAME_S),Linux)
  DIST_PLATFORM_OS := linux
else
  DIST_PLATFORM_OS := $(shell printf '%s' '$(UNAME_S)' | tr '[:upper:]' '[:lower:]')
endif

ifeq ($(UNAME_M),x86_64)
  DIST_PLATFORM_ARCH := amd64
else ifeq ($(UNAME_M),amd64)
  DIST_PLATFORM_ARCH := amd64
else ifeq ($(UNAME_M),aarch64)
  DIST_PLATFORM_ARCH := arm64
else ifeq ($(UNAME_M),arm64)
  DIST_PLATFORM_ARCH := arm64
else
  DIST_PLATFORM_ARCH := $(UNAME_M)
endif

DIST_PACKAGE_BASENAME ?= $(if $(strip $(ASSET_BASENAME)),$(ASSET_BASENAME),ctmd-$(VERSION)-$(DIST_PLATFORM_OS)-$(DIST_PLATFORM_ARCH))
DIST_PACKAGE_STAGING_DIR := $(RELEASEDIR)/$(DIST_PACKAGE_BASENAME)
DIST_PACKAGE_TARBALL := $(RELEASEDIR)/$(DIST_PACKAGE_BASENAME).tar.gz

.PHONY: dist verify-dist package-dist

dist: rust-sp1
	@echo "📦  Assembling dist layout in $(DISTDIR)"
	@rm -rf "$(DISTDIR)"
	@mkdir -p "$(DIST_BINDIR)" "$(DIST_LIBDIR)"

	@echo "🏗️  Building $(CTM_BINARY) -> $(DIST_BINDIR)/$(CTM_BINARY)"
	@cd $(CTM_DIR) && CGO_ENABLED="1" \
		$(GO) build $(GOFLAGS) $(BUILD_FLAGS) -o $(DIST_BINDIR)/$(CTM_BINARY) $(CTM_MAIN_PKG)

	@echo "📦  Copying Rust FFI lib -> $(DIST_LIBDIR)/$(RUST_SP1_LIB_BASENAME)"
	@cp "$(RUST_SP1_LIB_BUILD)" "$(DIST_LIBDIR)/$(RUST_SP1_LIB_BASENAME)"

	@echo "🔧  Bundling + fixup"
	@$(BUNDLE_AND_FIXUP) \
		"$(DIST_BINDIR)/$(CTM_BINARY)" "$(DIST_LIBDIR)" \
		"$(DIST_DARWIN_RPATH)" '$(DIST_LINUX_RPATH)'

	@echo "✅  Dist bundle ready: $(DISTDIR)"

verify-dist: dist
	@echo "🔍 Verifying dist bundle in $(DISTDIR)"
	@test -x "$(DIST_BINDIR)/$(CTM_BINARY)"
	@test -d "$(DIST_LIBDIR)"
	@test -f "$(DIST_LIBDIR)/$(RUST_SP1_LIB_BASENAME)"
	@if [ "$(ROCKSDB_ENABLED)" = "true" ]; then \
		set -- "$(DIST_LIBDIR)"/librocksdb*; \
		test -e "$$1"; \
	fi
	@echo "🔐  Recording binary checksum"
	@"$(ROOT)/scripts/bundle/checksum_libs.sh" "$(DIST_BINDIR)" "$(DISTDIR)/bin-checksums.sha256"
	@cat "$(DISTDIR)/bin-checksums.sha256"
	@echo "🔐  Recording lib checksums"
	@"$(ROOT)/scripts/bundle/checksum_libs.sh" "$(DIST_LIBDIR)" "$(DISTDIR)/lib-checksums.sha256"
	@cat "$(DISTDIR)/lib-checksums.sha256"
	@echo "✅  Dist bundle verified"

package-dist: verify-dist
	@echo "📦  Packaging $(DIST_PACKAGE_TARBALL)"
	@if [ "$(UNAME_S)" = "Darwin" ]; then export COPYFILE_DISABLE=1; fi; \
		rm -rf "$(DIST_PACKAGE_STAGING_DIR)" "$(DIST_PACKAGE_TARBALL)"; \
		mkdir -p "$(RELEASEDIR)" "$(DIST_PACKAGE_STAGING_DIR)"; \
		cp -R "$(DIST_BINDIR)" "$(DIST_PACKAGE_STAGING_DIR)/"; \
		cp -R "$(DIST_LIBDIR)" "$(DIST_PACKAGE_STAGING_DIR)/"; \
		if [ -f "$(DISTDIR)/bin-checksums.sha256" ]; then \
			cp "$(DISTDIR)/bin-checksums.sha256" "$(DIST_PACKAGE_STAGING_DIR)/"; \
		fi; \
		if [ -f "$(DISTDIR)/lib-checksums.sha256" ]; then \
			cp "$(DISTDIR)/lib-checksums.sha256" "$(DIST_PACKAGE_STAGING_DIR)/"; \
		fi; \
		tar -C "$(RELEASEDIR)" -czf "$(DIST_PACKAGE_TARBALL)" "$(DIST_PACKAGE_BASENAME)"; \
		rm -rf "$(DIST_PACKAGE_STAGING_DIR)"
	@echo "✅  Packaged: $(DIST_PACKAGE_TARBALL)"
