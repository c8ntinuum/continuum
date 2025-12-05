#!/usr/bin/make -f

###############################################################################
###                           Module & Versioning                           ###
###############################################################################

VERSION ?= $(shell echo $(shell git describe --tags --always) | sed 's/^v//')
TMVERSION := $(shell go list -m github.com/cometbft/cometbft | sed 's:.* ::')
COMMIT := $(shell git log -1 --format='%H')
LEDGER_ENABLED ?= true
ROCKSDB_ENABLED ?= true

###############################################################################
###                          Directories & Binaries                         ###
###############################################################################

BINDIR ?= $(GOPATH)/bin
BUILDDIR ?= $(CURDIR)/build
CTM_BINARY := ctmd
LINUX_LIBDIR ?= /usr/local/lib

###############################################################################
###                              Repo Info                                  ###
###############################################################################

HTTPS_GIT := https://github.com/c8ntinuum/continuum.git
DOCKER := $(shell which docker)
ifndef DOCKER
  $(warning docker not found in PATH; docker-based targets \
    (proto-*, localnet-*, release, test-rpc-compat, contracts-*) will not work)
endif

export GO111MODULE = on

###############################################################################
###                            Submodule Settings                           ###
###############################################################################

EVMD_DIR      := evmd
EVMD_MAIN_PKG := ./cmd/evmd

###############################################################################
###                        OS Detection.                                    ###
###############################################################################

UNAME_S := $(shell uname -s)

###############################################################################
###                         Rust SP1 Verifier                               ###
###############################################################################

RUST_SP1_DIR        := $(CURDIR)/rust/sp1verifier
ifeq ($(CARGO_BUILD_TARGET),)
  RUST_SP1_TARGET_DIR := $(RUST_SP1_DIR)/target/release
else
  RUST_SP1_TARGET_DIR := $(RUST_SP1_DIR)/target/$(CARGO_BUILD_TARGET)/release
endif
RUST_SP1_LIB_NAME   := sp1verifier
ifeq ($(UNAME_S),Darwin)
  DYNAMIC_LIB_EXT := dylib
else
  DYNAMIC_LIB_EXT := so
endif
RUST_SP1_LIB_BUILD    := $(RUST_SP1_TARGET_DIR)/lib$(RUST_SP1_LIB_NAME).$(DYNAMIC_LIB_EXT)
RUST_SP1_LIB_BASENAME := lib$(RUST_SP1_LIB_NAME).$(DYNAMIC_LIB_EXT)

###############################################################################
###                    RocksDB / CGO flags                                  ###
###############################################################################

ifeq ($(ROCKSDB_ENABLED),true)
  ROCKSDB_DETECTED := no
  
  ifeq ($(UNAME_S),Darwin)
    ROCKSDB_PREFIX ?= $(shell brew --prefix rocksdb 2>/dev/null)
    ZSTD_PREFIX    ?= $(shell brew --prefix zstd 2>/dev/null)
    SNAPPY_PREFIX  ?= $(shell brew --prefix snappy 2>/dev/null)
    LZ4_PREFIX     ?= $(shell brew --prefix lz4 2>/dev/null)
    BZ2_PREFIX     ?= $(shell brew --prefix bzip2 2>/dev/null)
    ZLIB_PREFIX    ?= $(shell brew --prefix zlib 2>/dev/null)
  endif

  ifeq ($(UNAME_S),Linux)
    ROCKSDB_PKG_CONFIG := $(shell command -v pkg-config >/dev/null 2>&1 && pkg-config --exists rocksdb 2>/dev/null && echo yes || echo no)
    ifeq ($(ROCKSDB_PKG_CONFIG),yes)
      CGO_CFLAGS  += $(shell pkg-config --cflags rocksdb 2>/dev/null)
      CGO_LDFLAGS += $(shell pkg-config --libs-only-L rocksdb 2>/dev/null)
      ROCKSDB_DETECTED := yes
    else
      ifeq ($(ROCKSDB_PREFIX),)
        ifneq ($(wildcard /usr/include/rocksdb/c.h),)
          ROCKSDB_PREFIX := /usr
        else ifneq ($(wildcard /usr/local/include/rocksdb/c.h),)
          ROCKSDB_PREFIX := /usr/local
        endif
      endif
    endif
  endif
  
  ifneq ($(ROCKSDB_PREFIX),)
    ifneq ($(wildcard $(ROCKSDB_PREFIX)/include/rocksdb/c.h),)
      CGO_CFLAGS  += -I$(ROCKSDB_PREFIX)/include
      ROCKSDB_DETECTED := yes
    endif
    ifneq ($(wildcard $(ROCKSDB_PREFIX)/lib/librocksdb.*),)
      CGO_LDFLAGS += -L$(ROCKSDB_PREFIX)/lib
    endif
  endif

  ifneq ($(ZSTD_PREFIX),)
    ifneq ($(wildcard $(ZSTD_PREFIX)/lib),)
      CGO_LDFLAGS += -L$(ZSTD_PREFIX)/lib
    endif
  endif
  ifneq ($(SNAPPY_PREFIX),)
    ifneq ($(wildcard $(SNAPPY_PREFIX)/lib),)
      CGO_LDFLAGS += -L$(SNAPPY_PREFIX)/lib
    endif
  endif
  ifneq ($(LZ4_PREFIX),)
    ifneq ($(wildcard $(LZ4_PREFIX)/lib),)
      CGO_LDFLAGS += -L$(LZ4_PREFIX)/lib
    endif
  endif
  ifneq ($(BZ2_PREFIX),)
    ifneq ($(wildcard $(BZ2_PREFIX)/lib),)
      CGO_LDFLAGS += -L$(BZ2_PREFIX)/lib
    endif
  endif
  ifneq ($(ZLIB_PREFIX),)
    ifneq ($(wildcard $(ZLIB_PREFIX)/lib),)
      CGO_LDFLAGS += -L$(ZLIB_PREFIX)/lib
    endif
  endif

  ifeq ($(ROCKSDB_DETECTED),yes)
    $(info Using RocksDB backend (ROCKSDB_ENABLED=true))
  else
    $(warning ROCKSDB_ENABLED=true but RocksDB headers/libs not found; falling back to goleveldb backend. Set ROCKSDB_ENABLED=false to silence this.)
    ROCKSDB_ENABLED := false
  endif
endif

###############################################################################
###                        Build & Install ctmd                             ###
###############################################################################

build_tags = netgo

# Ledger support
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
  build_tags += gcc
  build_tags += rocksdb
endif
build_tags += $(BUILD_TAGS)
build_tags := $(strip $(build_tags))

# process linker flags
ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=os \
          -X github.com/cosmos/cosmos-sdk/version.AppName=$(CTM_BINARY) \
          -X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
          -X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \
          -X github.com/cometbft/cometbft/version.TMCoreSemVer=$(TMVERSION)

# DB backend selection
ifeq (cleveldb,$(findstring cleveldb,$(COSMOS_BUILD_OPTIONS)))
  ldflags += -X github.com/cosmos/cosmos-sdk/types.DBBackend=cleveldb
endif
ifeq ($(ROCKSDB_ENABLED),true)
  ldflags += -X github.com/cosmos/cosmos-sdk/types.DBBackend=rocksdb
endif

# add build tags to linker flags
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
# check for nostrip option
ifeq (,$(findstring nostrip,$(COSMOS_BUILD_OPTIONS)))
  BUILD_FLAGS += -trimpath
endif

# check if no optimization option is passed
# used for remote debugging
ifneq (,$(findstring nooptimization,$(COSMOS_BUILD_OPTIONS)))
  BUILD_FLAGS += -gcflags "all=-N -l"
endif

###############################################################################
###                           Rust build targets                            ###
###############################################################################

.PHONY: rust-sp1
rust-sp1: $(RUST_SP1_LIB_BUILD)

$(RUST_SP1_LIB_BUILD):
	@echo "🦀  Building Rust SP1 verifier (native) ..."
	@cd $(RUST_SP1_DIR) && cargo build --release
	@if [ "$(UNAME_S)" = "Darwin" ]; then \
		echo "🔧  Setting install_name of libsp1verifier.dylib to @loader_path/$(RUST_SP1_LIB_BASENAME)"; \
		install_name_tool -id @loader_path/$(RUST_SP1_LIB_BASENAME) $(RUST_SP1_LIB_BUILD); \
	fi

###############################################################################
###                           Go build targets                              ###
###############################################################################

build: rust-sp1 go.sum $(BUILDDIR)/
	@echo "🏗️  Building ctmd to $(BUILDDIR)/$(CTM_BINARY) ..."
	@echo "BUILD_FLAGS: $(BUILD_FLAGS)"
	@mkdir -p $(BUILDDIR)
	@cp $(RUST_SP1_LIB_BUILD) $(BUILDDIR)/$(RUST_SP1_LIB_BASENAME)
	@cd $(EVMD_DIR) && CGO_ENABLED="1" \
	  go build $(BUILD_FLAGS) -o $(BUILDDIR)/$(CTM_BINARY) $(EVMD_MAIN_PKG)
	@if [ "$(UNAME_S)" = "Linux" ]; then \
		if command -v patchelf >/dev/null 2>&1; then \
			echo "🔧  Setting RPATH of $(BUILDDIR)/$(CTM_BINARY) to '\$$ORIGIN' for libsp1verifier.so"; \
			patchelf --set-rpath '$$ORIGIN' $(BUILDDIR)/$(CTM_BINARY); \
		else \
			echo "ℹ  patchelf not found; ensure libsp1verifier.so is on your loader path (LD_LIBRARY_PATH or system lib dir)."; \
		fi; \
	fi

install: rust-sp1 go.sum
	@echo "🚚  Installing $(CTM_BINARY) to $(BINDIR) ..."
	@echo "BUILD_FLAGS: $(BUILD_FLAGS)"
	@mkdir -p $(BINDIR)
	@cd $(EVMD_DIR) && CGO_ENABLED="1" \
	  go build $(BUILD_FLAGS) -o $(BINDIR)/$(CTM_BINARY) $(EVMD_MAIN_PKG)
	@if [ "$(UNAME_S)" = "Darwin" ]; then \
		echo "📦  Installing libsp1verifier.dylib next to binary in $(BINDIR)"; \
		cp $(RUST_SP1_LIB_BUILD) $(BINDIR)/$(RUST_SP1_LIB_BASENAME); \
	elif [ "$(UNAME_S)" = "Linux" ]; then \
		echo "📦  Installing libsp1verifier.so to $(LINUX_LIBDIR) (sudo may be required)"; \
		install -d $(LINUX_LIBDIR); \
		install -m 0755 $(RUST_SP1_LIB_BUILD) $(LINUX_LIBDIR)/$(RUST_SP1_LIB_BASENAME); \
		if command -v ldconfig >/dev/null 2>&1; then \
			echo "🔧  Running ldconfig"; \
			ldconfig || true; \
		else \
			echo "ℹ️  ldconfig not found; if your system uses a linker cache, run it manually."; \
		fi; \
	fi

$(BUILDDIR)/:
	mkdir -p $(BUILDDIR)/

###############################################################################
###                               Cleaning                                   ###
###############################################################################

.PHONY: clean
clean:
	@echo "🧹  Cleaning build artifacts..."
	@rm -rf $(BUILDDIR)
	@cd $(RUST_SP1_DIR) && cargo clean
	@cd $(EVMD_DIR) && go clean ./...

# Default & all target
.PHONY: all build install
all: build

###############################################################################
###                             Environment check                           ###
###############################################################################

doctor:
	@echo "🔍 Environment check"
	@if ! command -v go >/dev/null 2>&1; then \
		echo "⚠️  go not found; needed for build/test"; \
	fi
	@if ! command -v cargo >/dev/null 2>&1; then \
		echo "⚠️  cargo not found; needed for rust-sp1 verifier"; \
	fi
	@if ! command -v docker >/dev/null 2>&1; then \
		echo "⚠️  docker not found; docker-based targets (proto-*, localnet-*, release, test-rpc-compat, contracts-*) will fail"; \
	fi
	@if [ "$(UNAME_S)" = "Linux" ]; then \
		if ! command -v patchelf >/dev/null 2>&1; then \
			echo "ℹ️  patchelf not found; on Linux we won't auto-set RPATH for libsp1verifier.so (you must ensure it is on the loader path)"; \
		fi; \
	fi
	@if [ "$(UNAME_S)" = "Darwin" ]; then \
		echo "🔎 Checking RocksDB on Darwin ..."; \
		if brew --prefix rocksdb >/dev/null 2>&1; then \
			prefix=$$(brew --prefix rocksdb); \
			if [ -f "$$prefix/include/rocksdb/c.h" ]; then \
				echo "✅  rocksdb headers found at $$prefix/include"; \
			else \
				echo "⚠️  rocksdb via Homebrew found at $$prefix, but include/rocksdb/c.h is missing"; \
			fi; \
			if [ -d "$$prefix/lib" ]; then \
				echo "✅  rocksdb libs directory: $$prefix/lib"; \
			else \
				echo "⚠️  rocksdb libs directory $$prefix/lib not found"; \
			fi; \
		else \
			echo "ℹ️  rocksdb not found via Homebrew; COSMOS_BUILD_OPTIONS=rocksdb will fall back to goleveldb unless you set ROCKSDB_PREFIX manually"; \
		fi; \
	elif [ "$(UNAME_S)" = "Linux" ]; then \
		echo "🔎 Checking RocksDB (Linux) ..."; \
		if command -v pkg-config >/dev/null 2>&1 && pkg-config --exists rocksdb 2>/dev/null; then \
			echo "✅  rocksdb found via pkg-config"; \
		elif [ -f /usr/include/rocksdb/c.h ] || [ -f /usr/local/include/rocksdb/c.h ]; then \
			echo "✅  rocksdb headers found in /usr or /usr/local"; \
		else \
			echo "ℹ️  rocksdb headers not found in standard locations; COSMOS_BUILD_OPTIONS=rocksdb will fall back to goleveldb unless you set ROCKSDB_PREFIX"; \
		fi; \
	fi

.PHONY: doctor

###############################################################################
###                          Tools & Dependencies                           ###
###############################################################################

go.sum: go.mod
	echo "Ensure dependencies have not been modified ..." >&2
	go mod verify
	go mod tidy

vulncheck:
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@govulncheck ./...

###############################################################################
###                           Tests & Simulation                            ###
###############################################################################

PACKAGES_NOSIMULATION=$(shell go list ./... | grep -v '/simulation')
PACKAGES_UNIT := $(shell go list ./... | grep -v '/tests/e2e$$' | grep -v '/simulation')
PACKAGES_EVMD := $(shell cd evmd && go list ./... | grep -v '/simulation')
COVERPKG_EVM  := $(shell go list ./... | grep -v '/tests/e2e$$' | grep -v '/simulation' | paste -sd, -)
COVERPKG_ALL  := $(COVERPKG_EVM)
COMMON_COVER_ARGS := -timeout=15m -covermode=atomic

TEST_PACKAGES := ./...
TEST_TARGETS := test-unit test-evmd test-unit-cover test-race

test-unit: ARGS=-timeout=15m
test-unit: TEST_PACKAGES=$(PACKAGES_UNIT)
test-unit: run-tests

test-race: ARGS=-race
test-race: TEST_PACKAGES=$(PACKAGES_UNIT)
test-race: run-tests

test-evmd: ARGS=-timeout=15m
test-evmd:
	@cd evmd && go test -count=1 -race -tags=test -mod=readonly $(ARGS) $(EXTRA_ARGS) $(PACKAGES_EVMD)

test-unit-cover: ARGS=-timeout=15m -coverprofile=coverage.txt -covermode=atomic
test-unit-cover: TEST_PACKAGES=$(PACKAGES_UNIT)
test-unit-cover: run-tests
	@echo "🔍 Running evm (root) coverage..."
	@go test -race -tags=test $(COMMON_COVER_ARGS) -coverpkg=$(COVERPKG_ALL) -coverprofile=coverage.txt ./...
	@echo "🔍 Running evmd coverage..."
	@cd evmd && go test -race -tags=test $(COMMON_COVER_ARGS) -coverpkg=$(COVERPKG_ALL) -coverprofile=coverage_evmd.txt ./...
	@echo "🔀 Merging evmd coverage into root coverage..."
	@tail -n +2 evmd/coverage_evmd.txt >> coverage.txt && rm evmd/coverage_evmd.txt
	@echo "🧹 Filtering ignored files from coverage.txt..."
	@grep -v -E '/cmd/|/client/|/proto/|/testutil/|/mocks/|/test_.*\.go:|\.pb\.go:|\.pb\.gw\.go:|/x/[^/]+/module\.go:|/scripts/|/ibc/testing/|/version/|\.md:|\.pulsar\.go:' coverage.txt > tmp_coverage.txt && mv tmp_coverage.txt coverage.txt

test: test-unit

test-all:
	@echo "🔍 Running evm module tests..."
	@go test -race -tags=test -mod=readonly -timeout=15m $(PACKAGES_NOSIMULATION)
	@echo "🔍 Running evmd module tests..."
	@cd evmd && go test -race -tags=test -mod=readonly -timeout=15m $(PACKAGES_EVMD)

run-tests:
ifneq (,$(shell which tparse 2>/dev/null))
	go test -count=1 -race -tags=test -mod=readonly -json $(ARGS) $(EXTRA_ARGS) $(TEST_PACKAGES) | tparse
else
	go test -count=1 -race -tags=test -mod=readonly $(ARGS) $(EXTRA_ARGS) $(TEST_PACKAGES)
endif

# Use the old Apple linker to workaround broken xcode - https://github.com/golang/go/issues/65169
ifeq ($(OS_FAMILY),Darwin)
  FUZZLDFLAGS := -ldflags=-extldflags=-Wl,-ld_classic
endif

test-fuzz:
	go test -race -tags=test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzMintCoins ./x/precisebank/keeper
	go test -race -tags=test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzBurnCoins ./x/precisebank/keeper
	go test -race -tags=test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzSendCoins ./x/precisebank/keeper
	go test -race -tags=test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzGenesisStateValidate_NonZeroRemainder ./x/precisebank/types
	go test -race -tags=test $(FUZZLDFLAGS) -run NOTAREALTEST -v -fuzztime 10s -fuzz=FuzzGenesisStateValidate_ZeroRemainder ./x/precisebank/types

test-scripts:
	@echo "Running scripts tests"
	@pytest -s -vv ./scripts

test-solidity:
	@echo "Beginning solidity tests..."
	./scripts/run-solidity-tests.sh

.PHONY: run-tests test test-all $(TEST_TARGETS)

benchmark:
	@go test -race -tags=test -mod=readonly -bench=. $(PACKAGES_NOSIMULATION)

.PHONY: benchmark

###############################################################################
###                                Linting                                  ###
###############################################################################
golangci_lint_cmd=golangci-lint
golangci_version=v2.2.2

lint: lint-go lint-python lint-contracts

lint-go:
	@echo "--> Running linter"
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(golangci_version)
	@$(golangci_lint_cmd) run --timeout=15m

lint-python:
	find . -name "*.py" -type f -not -path "*/node_modules/*" | xargs pylint
	flake8

lint-contracts:
	solhint contracts/**/*.sol

lint-fix:
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(golangci_version)
	@$(golangci_lint_cmd) run --timeout=15m --fix

lint-fix-contracts:
	solhint --fix contracts/**/*.sol

.PHONY: lint lint-fix lint-contracts lint-go lint-python

format: format-go format-python format-shell

format-go:
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "./client/docs/statik/statik.go" -not -name '*.pb.go' -not -name '*.pb.gw.go' -not -name '*.pulsar.go' | xargs gofumpt -w -l

format-python: format-isort format-black

format-black:
	find . -name '*.py' -type f -not -path "*/node_modules/*" | xargs black

format-isort:
	find . -name '*.py' -type f -not -path "*/node_modules/*" | xargs isort

format-shell:
	shfmt -l -w .

.PHONY: format format-go format-python format-black format-isort

###############################################################################
###                                Protobuf                                 ###
###############################################################################

protoVer=0.14.0
protoImageName=ghcr.io/cosmos/proto-builder:$(protoVer)
protoImage=$(DOCKER) run --rm -v $(CURDIR):/workspace --workdir /workspace --user 0 $(protoImageName)

protoLintVer=0.44.0
protoLinterImage=yoheimuta/protolint
protoLinter=$(DOCKER) run --rm -v "$(CURDIR):/workspace" --workdir /workspace --user 0 $(protoLinterImage):$(protoLintVer)

# ------
# NOTE: If you are experiencing problems running these commands, try deleting
#       the docker images and execute the desired command again.
#
proto-all: proto-format proto-lint proto-gen

proto-gen:
	@echo "generating implementations from Protobuf files"
	@$(protoImage) sh ./scripts/generate_protos.sh
	@$(protoImage) sh ./scripts/generate_protos_pulsar.sh

proto-format:
	@echo "formatting Protobuf files"
	@$(protoImage) find ./ -name *.proto -exec clang-format -i {} \;

proto-lint:
	@echo "linting Protobuf files"
	@$(protoImage) buf lint --error-format=json
	@$(protoLinter) lint ./proto

proto-check-breaking:
	@echo "checking Protobuf files for breaking changes"
	@$(protoImage) buf breaking --against $(HTTPS_GIT)#branch=main

.PHONY: proto-all proto-gen proto-format proto-lint proto-check-breaking

###############################################################################
###                                Releasing                                ###
###############################################################################

PACKAGE_NAME:=github.com/c8ntinuum/continuum
GOLANG_CROSS_VERSION  = v1.22
GOPATH ?= $(HOME)/go
release-dry-run:
	docker run \
		--rm \
		--privileged \
		-e CGO_ENABLED=1 \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v `pwd`:/go/src/$(PACKAGE_NAME) \
		-v ${GOPATH}/pkg:/go/pkg \
		-w /go/src/$(PACKAGE_NAME) \
		ghcr.io/goreleaser/goreleaser-cross:${GOLANG_CROSS_VERSION} \
		--clean --skip validate --skip publish --snapshot

release:
	@if [ ! -f ".release-env" ]; then \
		echo "\033[91m.release-env is required for release\033[0m";\
		exit 1;\
	fi
	docker run \
		--rm \
		--privileged \
		-e CGO_ENABLED=1 \
		--env-file .release-env \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v `pwd`:/go/src/$(PACKAGE_NAME) \
		-w /go/src/$(PACKAGE_NAME) \
		ghcr.io/goreleaser/goreleaser-cross:${GOLANG_CROSS_VERSION} \
		release --clean --skip validate

.PHONY: release-dry-run release

###############################################################################
###                        Compile Solidity Contracts                       ###
###############################################################################

contracts-all: contracts-compile contracts-clean

contracts-clean:
	@echo "Cleaning up the contracts directory..."
	@python3 ./scripts/compile_smart_contracts/compile_smart_contracts.py --clean

contracts-compile:
	@echo "Compiling smart contracts..."
	@python3 ./scripts/compile_smart_contracts/compile_smart_contracts.py --compile

contracts-add:
	@echo "Adding a new smart contract to be compiled..."
	@python3 ./scripts/compile_smart_contracts/compile_smart_contracts.py --add $(CONTRACT)

###############################################################################
###                                Localnet                                 ###
###############################################################################

localnet-build-env:
	$(MAKE) -C contrib/images evmd-env

localnet-build-nodes:
	$(DOCKER) run --rm -v $(CURDIR)/.testnets:/data cosmos/evmd \
			  testnet init-files --validator-count 4 -o /data --starting-ip-address 192.168.10.2 --keyring-backend=test --chain-id=local-4221 --use-docker=true
	docker compose up -d

localnet-stop:
	docker compose down

# localnet-start will run a 4-node testnet locally. The nodes are
# based off the docker images in: ./contrib/images/simd-env
localnet-start: localnet-stop localnet-build-env localnet-build-nodes


test-rpc-compat:
	@./tests/jsonrpc/scripts/run-compat-test.sh

test-rpc-compat-stop:
	cd tests/jsonrpc && docker compose down

.PHONY: localnet-start localnet-stop localnet-build-env localnet-build-nodes test-rpc-compat test-rpc-compat-stop

test-system: build-v04 build
	mkdir -p ./tests/systemtests/binaries/
	cp $(BUILDDIR)/$(CTM_BINARY) ./tests/systemtests/binaries/
	cd tests/systemtests/Counter && forge build
	$(MAKE) -C tests/systemtests test

build-v04:
	mkdir -p ./tests/systemtests/binaries/v0.4
	git checkout v0.4.1
	make build
	cp $(BUILDDIR)/$(CTM_BINARY) ./tests/systemtests/binaries/v0.4
	git checkout -

mocks:
	@echo "--> generating mocks"
	@go get github.com/vektra/mockery/v2
	@go generate ./...
	@make format-go

###############################################################################
###                              D2 Diagrams                                ###
###############################################################################

D2_THEME=300
D2_DARK_THEME=200
D2_LAYOUT=tala

D2_ENV_VARS=D2_THEME=$(D2_THEME) \
	    D2_DARK_THEME=$(D2_DARK_THEME) \
	    D2_LAYOUT=$(D2_LAYOUT)

.PHONY: d2check d2watch d2gen d2gen-all

d2check:
	@echo "🔍 checking if d2 is installed..."
	@which d2 > /dev/null 2>&1 || { \
		echo "🔴 d2 is not installed, see installation docs: https://d2lang.com/tour/install/"; \
		exit 1; \
	}
	@echo "🟢 d2 is installed"
	@echo "🔍 checking if $(D2_LAYOUT) layout is installed..."
	@d2 layout | grep $(D2_LAYOUT) > /dev/null 2>&1 || { \
		echo "🔴 $(D2_LAYOUT) layout is not installed, see docs: https://d2lang.com/tour/layouts/"; \
		exit 1; \
	}
	@echo "🟢 $(D2_LAYOUT) layout is installed"

d2watch: d2check
	@if [ -z "$(FILE)" ]; then \
		echo "🔴 missing required parameter FILE, the correct usage is: make d2watch FILE=path/to/file.d2"; \
		exit 1; \
	fi
	@if [ ! -f "$(FILE)" ]; then \
		echo "🔴 file $(FILE) does not exist"; \
		exit 1; \
	fi
	@echo "🔄 watching $(FILE) for changes..."
	@dir=$$(dirname "$(FILE)"); \
	basename=$$(basename "$(FILE)" .d2); \
	svgfile="$$dir/$$basename.svg"; \
	printf "📊 generating $$svgfile from $(FILE)... "; \
	$(D2_ENV_VARS) d2 --watch "$(FILE)" "$$svgfile"

d2gen: d2check
	@if [ -z "$(FILE)" ]; then \
		echo "🔴 missing required parameter FILE, the correct usage is: make d2gen FILE=path/to/file.d2"; \
		exit 1; \
	fi
	@if [ ! -f "$(FILE)" ]; then \
		echo "🔴 file $(FILE) does not exist"; \
		exit 1; \
	fi
	@dir=$$(dirname "$(FILE)"); \
	basename=$$(basename "$(FILE)" .d2); \
	svgfile="$$dir/$$basename.svg"; \
	printf "📊 generating $$svgfile from $(FILE)... "; \
	$(D2_ENV_VARS) d2 "$(FILE)" "$$svgfile" > /dev/null 2>&1 && echo "done ✅" || echo "failed ❌";

d2gen-all: d2check
	@echo "🟢 generating svg files for all d2 diagrams..."
	@find . -name "*.d2" -type f | while read d2file; do \
		dir=$$(dirname "$$d2file"); \
		basename=$$(basename "$$d2file" .d2); \
		svgfile="$$dir/$$basename.svg"; \
		printf "📊 generating $$svgfile from $$d2file... "; \
		$(D2_ENV_VARS) d2 "$$d2file" "$$svgfile" > /dev/null 2>&1 && echo "done ✅" || echo "failed ❌"; \
	done
	@echo "✅ svg files generated for all d2 diagrams"