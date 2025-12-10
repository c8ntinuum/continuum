###############################################################################
# Rust SP1 verifier: builds libsp1verifier.{so,dylib} for Go FFI/cgo usage
###############################################################################

ROOT    ?= $(CURDIR)
UNAME_S ?= $(shell uname -s)

CARGO ?= cargo
RUSTUP_TOOLCHAIN ?=

RUST_SP1_DIR      ?= $(ROOT)/rust/sp1verifier
RUST_SP1_MANIFEST ?= $(RUST_SP1_DIR)/Cargo.toml
RUST_SP1_LIB_NAME ?= sp1verifier
RUST_SP1_INCLUDE_DIR ?= $(RUST_SP1_DIR)/include

# release|dev|<custom profile>
RUST_SP1_PROFILE ?= release

# optional cross target
RUST_SP1_TARGET ?= $(firstword $(CARGO_BUILD_TARGET))

RUST_SP1_TARGET_DIR_BASE ?= $(RUST_SP1_DIR)/target

RUST_SP1_FEATURES ?=
RUST_SP1_ALL_FEATURES ?= false
RUST_SP1_NO_DEFAULT_FEATURES ?= false
RUST_SP1_LOCKED ?= false
RUST_SP1_FROZEN ?= false
RUST_SP1_CARGO_FLAGS ?=

RUST_SP1_DARWIN_SET_INSTALL_NAME ?= true
RUST_SP1_INSTALL_NAME ?= @rpath/$(RUST_SP1_LIB_BASENAME)

ifneq ($(strip $(RUSTUP_TOOLCHAIN)),)
  RUST_SP1_CARGO := $(CARGO) +$(RUSTUP_TOOLCHAIN)
else
  RUST_SP1_CARGO := $(CARGO)
endif
RUST_SP1_CARGO_BIN := $(firstword $(CARGO))

ifeq ($(UNAME_S),Darwin)
  RUST_SP1_DYLIB_EXT := dylib
else ifeq ($(OS),Windows_NT)
  RUST_SP1_DYLIB_EXT := dll
else
  RUST_SP1_DYLIB_EXT := so
endif

RUST_SP1_LIB_BASENAME := lib$(RUST_SP1_LIB_NAME).$(RUST_SP1_DYLIB_EXT)

ifeq ($(RUST_SP1_PROFILE),release)
  RUST_SP1_PROFILE_FLAG := --release
  RUST_SP1_PROFILE_DIR  := release
else ifeq ($(RUST_SP1_PROFILE),dev)
  RUST_SP1_PROFILE_FLAG :=
  RUST_SP1_PROFILE_DIR  := debug
else
  RUST_SP1_PROFILE_FLAG := --profile $(RUST_SP1_PROFILE)
  RUST_SP1_PROFILE_DIR  := $(RUST_SP1_PROFILE)
endif

ifneq ($(strip $(RUST_SP1_TARGET)),)
  RUST_SP1_TARGET_FLAG := --target $(RUST_SP1_TARGET)
  RUST_SP1_OUT_DIR := $(RUST_SP1_TARGET_DIR_BASE)/$(RUST_SP1_TARGET)/$(RUST_SP1_PROFILE_DIR)
else
  RUST_SP1_TARGET_FLAG :=
  RUST_SP1_OUT_DIR := $(RUST_SP1_TARGET_DIR_BASE)/$(RUST_SP1_PROFILE_DIR)
endif

RUST_SP1_LIB_BUILD := $(RUST_SP1_OUT_DIR)/$(RUST_SP1_LIB_BASENAME)

RUST_SP1_FEATURE_ARGS :=
ifneq ($(strip $(RUST_SP1_FEATURES)),)
  RUST_SP1_FEATURE_ARGS += --features $(RUST_SP1_FEATURES)
endif
ifeq ($(RUST_SP1_ALL_FEATURES),true)
  RUST_SP1_FEATURE_ARGS += --all-features
endif
ifeq ($(RUST_SP1_NO_DEFAULT_FEATURES),true)
  RUST_SP1_FEATURE_ARGS += --no-default-features
endif

RUST_SP1_LOCK_ARGS :=
ifeq ($(RUST_SP1_FROZEN),true)
  RUST_SP1_LOCK_ARGS += --frozen
else ifeq ($(RUST_SP1_LOCKED),true)
  RUST_SP1_LOCK_ARGS += --locked
endif

RUST_SP1_DEPS := $(RUST_SP1_MANIFEST)
ifneq ($(wildcard $(RUST_SP1_DIR)/Cargo.lock),)
  RUST_SP1_DEPS += $(RUST_SP1_DIR)/Cargo.lock
endif
ifneq ($(wildcard $(RUST_SP1_DIR)/build.rs),)
  RUST_SP1_DEPS += $(RUST_SP1_DIR)/build.rs
endif
RUST_SP1_DEPS += $(shell find $(RUST_SP1_DIR)/src -type f -name '*.rs' 2>/dev/null)

RUST_SP1_CGO_CFLAGS  ?= -I$(RUST_SP1_INCLUDE_DIR)
RUST_SP1_CGO_LDFLAGS ?= -L$(RUST_SP1_OUT_DIR)

.PHONY: rust-sp1 rust-sp1-clean
rust-sp1: $(RUST_SP1_LIB_BUILD)

$(RUST_SP1_LIB_BUILD): $(RUST_SP1_DEPS)
	@command -v "$(RUST_SP1_CARGO_BIN)" >/dev/null 2>&1 || { \
		echo "ERROR: '$(RUST_SP1_CARGO_BIN)' not found (needed for rust/sp1verifier)."; \
		exit 1; \
	}
	@echo "🦀  Building Rust SP1 verifier ($(RUST_SP1_PROFILE)$(if $(RUST_SP1_TARGET),; target=$(RUST_SP1_TARGET),)) ..."
	@$(RUST_SP1_CARGO) build \
		--manifest-path "$(RUST_SP1_MANIFEST)" \
		--lib \
		$(RUST_SP1_PROFILE_FLAG) \
		$(RUST_SP1_TARGET_FLAG) \
		--target-dir "$(RUST_SP1_TARGET_DIR_BASE)" \
		$(RUST_SP1_FEATURE_ARGS) \
		$(RUST_SP1_LOCK_ARGS) \
		$(RUST_SP1_CARGO_FLAGS)
	@if [ ! -f "$@" ]; then \
		echo "ERROR: expected Rust dynamic library not found: $@"; \
		echo "Hint: ensure rust/sp1verifier/Cargo.toml builds a C-loadable dylib (crate-type = [\"cdylib\"])."; \
		exit 1; \
	fi
	@if [ "$(UNAME_S)" = "Darwin" ] && [ "$(RUST_SP1_DARWIN_SET_INSTALL_NAME)" = "true" ] && command -v install_name_tool >/dev/null 2>&1; then \
		echo "🔧  macOS: setting install_name id -> $(RUST_SP1_INSTALL_NAME)"; \
		install_name_tool -id "$(RUST_SP1_INSTALL_NAME)" "$@"; \
	fi

rust-sp1-clean:
	@echo "🧹  Cleaning Rust SP1 verifier artifacts under $(RUST_SP1_TARGET_DIR_BASE) ..."
	@rm -rf "$(RUST_SP1_TARGET_DIR_BASE)"
