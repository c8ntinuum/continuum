###############################################################################
# RocksDB detection + CGO export variables (does not mutate global CGO_*)
###############################################################################

UNAME_S ?= $(shell uname -s)
PKG_CONFIG ?= pkg-config
ROCKSDB_PKG_NAME ?= rocksdb

ROCKSDB_ENABLED ?= true
ROCKSDB_REQUIRED ?= false
ROCKSDB_USE_PKG_CONFIG ?= auto

ROCKSDB_HEADER ?= rocksdb/c.h

ROCKSDB_PREFIX ?=
ROCKSDB_INCLUDE_DIR ?=
ROCKSDB_LIB_DIR ?=
ROCKSDB_CFLAGS ?=
ROCKSDB_LDFLAGS ?=
ROCKSDB_LIBS ?= -lrocksdb

ROCKSDB_DETECTED := no
ROCKSDB_DETECT_REASON :=
ROCKSDB_CGO_CFLAGS :=
ROCKSDB_CGO_LDFLAGS :=

ifeq ($(UNAME_S),Darwin)
  ROCKSDB_PREFIX ?= $(shell command -v brew >/dev/null 2>&1 && brew --prefix rocksdb 2>/dev/null)
  ZSTD_PREFIX    ?= $(shell command -v brew >/dev/null 2>&1 && brew --prefix zstd 2>/dev/null)
  SNAPPY_PREFIX  ?= $(shell command -v brew >/dev/null 2>&1 && brew --prefix snappy 2>/dev/null)
  LZ4_PREFIX     ?= $(shell command -v brew >/dev/null 2>&1 && brew --prefix lz4 2>/dev/null)
  BZ2_PREFIX     ?= $(shell command -v brew >/dev/null 2>&1 && brew --prefix bzip2 2>/dev/null)
  ZLIB_PREFIX    ?= $(shell command -v brew >/dev/null 2>&1 && brew --prefix zlib 2>/dev/null)
  GFLAGS_PREFIX  ?= $(shell command -v brew >/dev/null 2>&1 && brew --prefix gflags 2>/dev/null)
endif

ifeq ($(UNAME_S),Linux)
  ifeq ($(strip $(ROCKSDB_PREFIX)),)
    ifneq ($(wildcard /usr/include/$(ROCKSDB_HEADER)),)
      ROCKSDB_PREFIX := /usr
    else ifneq ($(wildcard /usr/local/include/$(ROCKSDB_HEADER)),)
      ROCKSDB_PREFIX := /usr/local
    endif
  endif
endif

ifneq ($(strip $(ROCKSDB_PREFIX)),)
  ROCKSDB_INCLUDE_DIR ?= $(ROCKSDB_PREFIX)/include
endif

ROCKSDB__LIB_CANDIDATES :=
ifneq ($(strip $(ROCKSDB_PREFIX)),)
  ROCKSDB__LIB_CANDIDATES += \
    $(wildcard $(ROCKSDB_PREFIX)/lib/librocksdb.so) \
    $(wildcard $(ROCKSDB_PREFIX)/lib64/librocksdb.so) \
    $(wildcard $(ROCKSDB_PREFIX)/lib/*/librocksdb.so) \
    $(wildcard $(ROCKSDB_PREFIX)/lib64/*/librocksdb.so) \
    $(wildcard $(ROCKSDB_PREFIX)/lib/librocksdb.dylib) \
    $(wildcard $(ROCKSDB_PREFIX)/lib64/librocksdb.dylib) \
    $(wildcard $(ROCKSDB_PREFIX)/lib/*/librocksdb.dylib) \
    $(wildcard $(ROCKSDB_PREFIX)/lib64/*/librocksdb.dylib) \
    $(wildcard $(ROCKSDB_PREFIX)/lib/librocksdb.a) \
    $(wildcard $(ROCKSDB_PREFIX)/lib64/librocksdb.a) \
    $(wildcard $(ROCKSDB_PREFIX)/lib/*/librocksdb.a) \
    $(wildcard $(ROCKSDB_PREFIX)/lib64/*/librocksdb.a)
endif

ROCKSDB__LIB_FILE := $(firstword $(ROCKSDB__LIB_CANDIDATES))
ifneq ($(strip $(ROCKSDB__LIB_FILE)),)
  ROCKSDB_LIB_DIR ?= $(patsubst %/,%,$(dir $(ROCKSDB__LIB_FILE)))
endif

ROCKSDB__PKG_OK := no
ifneq ($(filter true auto,$(ROCKSDB_USE_PKG_CONFIG)),)
  ROCKSDB__PKG_OK := $(shell command -v $(PKG_CONFIG) >/dev/null 2>&1 && $(PKG_CONFIG) --exists $(ROCKSDB_PKG_NAME) 2>/dev/null && echo yes || echo no)
endif

ifeq ($(ROCKSDB_ENABLED),true)
  ifeq ($(ROCKSDB__PKG_OK),yes)
    ROCKSDB_CGO_CFLAGS  += $(strip $(shell $(PKG_CONFIG) --cflags-only-I $(ROCKSDB_PKG_NAME) 2>/dev/null))
    ROCKSDB_CGO_LDFLAGS += $(strip $(shell $(PKG_CONFIG) --libs   $(ROCKSDB_PKG_NAME) 2>/dev/null))
    ROCKSDB_DETECTED := yes
    ROCKSDB_DETECT_REASON := pkg-config

    ifeq ($(strip $(ROCKSDB_PREFIX)),)
      ROCKSDB_PREFIX := $(strip $(shell $(PKG_CONFIG) --variable=prefix $(ROCKSDB_PKG_NAME) 2>/dev/null))
    endif
  else
    ROCKSDB__HDR := $(ROCKSDB_INCLUDE_DIR)/$(ROCKSDB_HEADER)
    ifneq ($(wildcard $(ROCKSDB__HDR)),)
      ifneq ($(strip $(ROCKSDB_LIB_DIR)),)
        ROCKSDB_CGO_CFLAGS  += -I$(ROCKSDB_INCLUDE_DIR) $(ROCKSDB_CFLAGS)
        ROCKSDB_CGO_LDFLAGS += -L$(ROCKSDB_LIB_DIR) $(ROCKSDB_LDFLAGS)
        ROCKSDB_DETECTED := yes
        ROCKSDB_DETECT_REASON := prefix
      endif
    endif
  endif

  ifeq ($(ROCKSDB_DETECTED),yes)
    $(info Using RocksDB backend (via $(ROCKSDB_DETECT_REASON)))
  else
    ifeq ($(ROCKSDB_REQUIRED),true)
      $(error ROCKSDB_ENABLED=true but RocksDB not found. Install rocksdb dev packages or set ROCKSDB_PREFIX/ROCKSDB_INCLUDE_DIR/ROCKSDB_LIB_DIR; or set ROCKSDB_ENABLED=false.)
    else
      $(warning ROCKSDB_ENABLED=true but RocksDB not found; disabling and falling back to goleveldb. Set ROCKSDB_REQUIRED=true to hard fail.)
      ROCKSDB_ENABLED := false
    endif
  endif
endif

# On macOS, also add -L paths for RocksDB companion libs when brew prefixes are set.
# This ensures -lzstd, -lsnappy, etc. can be resolved.
ifeq ($(UNAME_S),Darwin)
  ifneq ($(strip $(ZSTD_PREFIX)),)
    ROCKSDB_CGO_LDFLAGS += -L$(ZSTD_PREFIX)/lib
  endif
  ifneq ($(strip $(SNAPPY_PREFIX)),)
    ROCKSDB_CGO_LDFLAGS += -L$(SNAPPY_PREFIX)/lib
  endif
  ifneq ($(strip $(LZ4_PREFIX)),)
    ROCKSDB_CGO_LDFLAGS += -L$(LZ4_PREFIX)/lib
  endif
  ifneq ($(strip $(BZ2_PREFIX)),)
    ROCKSDB_CGO_LDFLAGS += -L$(BZ2_PREFIX)/lib
  endif
  ifneq ($(strip $(ZLIB_PREFIX)),)
    ROCKSDB_CGO_LDFLAGS += -L$(ZLIB_PREFIX)/lib
  endif
  ifneq ($(strip $(GFLAGS_PREFIX)),)
    ROCKSDB_CGO_LDFLAGS += -L$(GFLAGS_PREFIX)/lib
  endif
endif

ROCKSDB_BUNDLE_LIBS ?= rocksdb snappy zstd lz4 bz2 z gflags numa
