###############################################################################
# Go build/test targets + BUILD_FLAGS assembly
###############################################################################

UNAME_S ?= $(shell uname -s)
GO ?= go
GOFLAGS ?=

# Compose and export CGO flags for Go builds
CGO_CFLAGS  += $(ROCKSDB_CGO_CFLAGS)  $(RUST_SP1_CGO_CFLAGS)
CGO_LDFLAGS += $(ROCKSDB_CGO_LDFLAGS) $(RUST_SP1_CGO_LDFLAGS)
export CGO_CFLAGS
export CGO_LDFLAGS

# Build dir helper
$(BUILDDIR)/:
	@mkdir -p "$(BUILDDIR)/"

# Build tags
build_tags = netgo

ifeq ($(LEDGER_ENABLED),true)
  ifeq ($(OS),Windows_NT)
    GCCEXE = $(shell where gcc.exe 2> NUL)
    ifeq ($(GCCEXE),)
      $(error gcc.exe not installed for ledger support, please install or set LEDGER_ENABLED=false)
    else
      build_tags += ledger
    endif
  else
    ifeq ($(UNAME_S),OpenBSD)
      $(warning OpenBSD detected, disabling ledger support (https://github.com/cosmos/cosmos-sdk/issues/1988))
    else
      GCC = $(shell command -v gcc 2> /dev/null)
      ifeq ($(GCC),)
        $(error gcc not installed for ledger support, please install or set LEDGER_ENABLED=false)
      else
        build_tags += ledger
      endif
    endif
  endif
endif

ifeq (cleveldb,$(findstring cleveldb,$(COSMOS_BUILD_OPTIONS)))
  build_tags += gcc
endif
ifeq ($(ROCKSDB_ENABLED),true)
  build_tags += gcc rocksdb
endif

build_tags += $(BUILD_TAGS)
build_tags := $(strip $(build_tags))

# Linker flags
ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=os \
          -X github.com/cosmos/cosmos-sdk/version.AppName=$(CTM_BINARY) \
          -X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
          -X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \
          -X github.com/cometbft/cometbft/version.TMCoreSemVer=$(TMVERSION)

ifeq (cleveldb,$(findstring cleveldb,$(COSMOS_BUILD_OPTIONS)))
  ldflags += -X github.com/cosmos/cosmos-sdk/types.DBBackend=cleveldb
endif
ifeq ($(ROCKSDB_ENABLED),true)
  ldflags += -X github.com/cosmos/cosmos-sdk/types.DBBackend=rocksdb
endif

whitespace := $(subst ,, )
comma := ,
build_tags_comma_sep := $(subst $(whitespace),$(comma),$(build_tags))
ldflags += -X "github.com/cosmos/cosmos-sdk/version.BuildTags=$(build_tags_comma_sep)"

ifeq (,$(findstring nostrip,$(COSMOS_BUILD_OPTIONS)))
  ldflags += -w -s
endif
ldflags += $(LDFLAGS)
ldflags := $(strip $(ldflags))

ifeq (staticlink,$(findstring staticlink,$(COSMOS_BUILD_OPTIONS)))
  ldflags += -linkmode external -extldflags '-static'
endif

BUILD_FLAGS := -tags "$(build_tags)" -ldflags '$(ldflags)'

ifeq (,$(findstring nostrip,$(COSMOS_BUILD_OPTIONS)))
  BUILD_FLAGS += -trimpath
endif
ifneq (,$(findstring nooptimization,$(COSMOS_BUILD_OPTIONS)))
  BUILD_FLAGS += -gcflags "all=-N -l"
endif

.PHONY: all build build-go test fmt vet tidy mod-download print-go-env
all: build

build-go: $(BUILDDIR)/
	@echo "🏗️  Building $(CTM_BINARY) -> $(BUILDDIR)/$(CTM_BINARY)"
	@echo "BUILD_FLAGS: $(BUILD_FLAGS)"
	@cd $(EVMD_DIR) && CGO_ENABLED="1" \
		$(GO) build $(GOFLAGS) $(BUILD_FLAGS) -o $(BUILDDIR)/$(CTM_BINARY) $(EVMD_MAIN_PKG)

# build: dev build
build: rust-sp1 build-go
	@if [ "$(UNAME_S)" = "Darwin" ]; then \
		echo "📦  Dev (Darwin): placing $(RUST_SP1_LIB_BASENAME) next to $(BUILDDIR)/$(CTM_BINARY)"; \
		cp "$(RUST_SP1_LIB_BUILD)" "$(BUILDDIR)/$(RUST_SP1_LIB_BASENAME)"; \
		SP1_DEP=$$(otool -L "$(BUILDDIR)/$(CTM_BINARY)" | awk '/libsp1verifier/ {print $$1; exit}'); \
		if [ -n "$$SP1_DEP" ]; then \
			echo "🔧  Rewriting SP1 dep $$SP1_DEP -> @executable_path/$(RUST_SP1_LIB_BASENAME)"; \
			install_name_tool -change "$$SP1_DEP" "@executable_path/$(RUST_SP1_LIB_BASENAME)" "$(BUILDDIR)/$(CTM_BINARY)" || true; \
		fi; \
		echo "🔧  Setting ID of local $(RUST_SP1_LIB_BASENAME) to @executable_path/$(RUST_SP1_LIB_BASENAME)"; \
		install_name_tool -id "@executable_path/$(RUST_SP1_LIB_BASENAME)" "$(BUILDDIR)/$(RUST_SP1_LIB_BASENAME)" || true; \
	elif [ "$(UNAME_S)" = "Linux" ]; then \
		echo "📦  Dev (Linux): copying $(RUST_SP1_LIB_BASENAME) next to $(BUILDDIR)/$(CTM_BINARY)"; \
		cp "$(RUST_SP1_LIB_BUILD)" "$(BUILDDIR)/$(RUST_SP1_LIB_BASENAME)"; \
		if command -v patchelf >/dev/null 2>&1; then \
			echo "🔧  Setting RPATH of $(CTM_BINARY) to '\$$ORIGIN'"; \
			patchelf --force-rpath --set-rpath '$$ORIGIN' "$(BUILDDIR)/$(CTM_BINARY)" || true; \
		else \
			echo "ℹ️  patchelf not found; run ctmd with:"; \
			echo "    LD_LIBRARY_PATH=$(BUILDDIR) ./ctmd"; \
		fi; \
	fi
	@echo "✅  Build complete: $(BUILDDIR)/$(CTM_BINARY)"

GO_TEST_FLAGS ?= -count=1
GO_PKGS ?= ./...

test:
	@cd $(EVMD_DIR) && CGO_ENABLED="1" \
		$(GO) test $(GOFLAGS) -tags "$(build_tags)" $(GO_TEST_FLAGS) $(GO_PKGS)

fmt:
	@cd $(EVMD_DIR) && $(GO) fmt $(GO_PKGS)

vet:
	@cd $(EVMD_DIR) && $(GO) vet -tags "$(build_tags)" $(GO_PKGS)

mod-download:
	@cd $(EVMD_DIR) && $(GO) mod download

tidy:
	@cd $(EVMD_DIR) && $(GO) mod tidy

print-go-env:
	@echo "EVMD_DIR=$(EVMD_DIR)"
	@echo "EVMD_MAIN_PKG=$(EVMD_MAIN_PKG)"
	@echo "build_tags=$(build_tags)"
	@echo "BUILD_FLAGS=$(BUILD_FLAGS)"
	@echo "CGO_CFLAGS=$(CGO_CFLAGS)"
	@echo "CGO_LDFLAGS=$(CGO_LDFLAGS)"
