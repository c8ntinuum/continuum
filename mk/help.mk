.PHONY: help
help:
	@echo "Targets:"
	@echo "  build        Build ctmd (dev) + Rust lib"
	@echo "  build-go     Build ctmd only (Go)"
	@echo "  dist         Build a self-contained bundle in build/dist"
	@echo "  install      Install into BINDIR (+ Frameworks/ or lib/ bundle)"
	@echo "  install-system  (Linux) install Rust lib into /usr/local/lib (sudo required)"
	@echo "  test fmt vet tidy mod-download clean doctor"
	@echo ""
	@echo "Common overrides:"
	@echo "  ROCKSDB_ENABLED=true|false, ROCKSDB_REQUIRED=true"
	@echo "  ROCKSDB_PREFIX=..., PKG_CONFIG=..."
	@echo "  RUST_SP1_PROFILE=release|dev"
	@echo "  GOFLAGS=..., BINDIR=..., BUILDDIR=..."
	