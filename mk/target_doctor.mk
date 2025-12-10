UNAME_S ?= $(shell uname -s)

.PHONY: doctor
doctor:
	@echo "🔍 Environment check"
	@command -v go >/dev/null 2>&1 || echo "⚠️  go not found"
	@command -v cargo >/dev/null 2>&1 || echo "⚠️  cargo not found"
	@if [ "$(UNAME_S)" = "Darwin" ]; then \
	  command -v otool >/dev/null 2>&1 || echo "⚠️  otool missing"; \
	  command -v install_name_tool >/dev/null 2>&1 || echo "⚠️  install_name_tool missing"; \
	elif [ "$(UNAME_S)" = "Linux" ]; then \
	  command -v ldd >/dev/null 2>&1 || echo "⚠️  ldd missing"; \
	  command -v patchelf >/dev/null 2>&1 || echo "⚠️  patchelf missing (dist/install bundles may rely on LD_LIBRARY_PATH)"; \
	fi
	@test -x "$(ROOT)/scripts/bundle/bundle_fixup.sh" || echo "⚠️  scripts/bundle/bundle_fixup.sh not executable"
	@test -x "$(ROOT)/scripts/bundle/darwin/bundle_libs.sh" || echo "⚠️  scripts/bundle/darwin/bundle_libs.sh not executable"
	@test -x "$(ROOT)/scripts/bundle/darwin/fixup_bundle.sh" || echo "⚠️  scripts/bundle/darwin/fixup_bundle.sh not executable"
	@test -x "$(ROOT)/scripts/bundle/linux/fixup_bundle.sh" || echo "⚠️  scripts/bundle/linux/fixup_bundle.sh not executable"
	@test -x "$(ROOT)/scripts/bundle/linux/fixup_bundle.sh" || echo "⚠️  scripts/bundle/linux/fixup_bundle.sh not executable"
	@test -x "$(ROOT)/scripts/bundle/bundle_fixup.sh" || echo "⚠️  scripts/bundle/bundle_fixup.sh not executable"
