###############################################################################
# Shared bundling glue
###############################################################################

ROOT ?= $(CURDIR)

BUNDLE_AND_FIXUP ?= $(ROOT)/scripts/bundle/bundle_fixup.sh

ROCKSDB_BUNDLE_LIBS ?= rocksdb snappy zstd lz4 bz2 z gflags numa

DIST_DARWIN_RPATH   ?= @executable_path/../lib
DIST_LINUX_RPATH    ?= $$ORIGIN/../lib

INSTALL_DARWIN_RPATH ?= @executable_path/Frameworks
INSTALL_LINUX_RPATH  ?= $$ORIGIN/lib

export ROCKSDB_ENABLED
export ROCKSDB_BUNDLE_LIBS

export ROCKSDB_PREFIX ROCKSDB_LIB_DIR
export SNAPPY_PREFIX ZSTD_PREFIX LZ4_PREFIX BZ2_PREFIX ZLIB_PREFIX GFLAGS_PREFIX
