#!/usr/bin/make -f

ROOT := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))

-include $(ROOT)/local.mk

.DEFAULT_GOAL := build

include $(ROOT)/mk/core.mk
include $(ROOT)/mk/rust_sp1.mk
include $(ROOT)/mk/rocksdb.mk
include $(ROOT)/mk/go_build.mk
include $(ROOT)/mk/bundle.mk
include $(ROOT)/mk/target_dist.mk
include $(ROOT)/mk/target_install.mk
include $(ROOT)/mk/target_clean.mk
include $(ROOT)/mk/target_doctor.mk
include $(ROOT)/mk/help.mk
