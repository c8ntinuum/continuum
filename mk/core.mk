###############################################################################
###                           Module & Versioning                           ###
###############################################################################

VERSION ?= $(shell echo $(shell git describe --tags --always 2>/dev/null) | sed 's/^v//' )
TMVERSION := $(shell go list -m github.com/cometbft/cometbft 2>/dev/null | sed 's:.* ::')
COMMIT := $(shell git log -1 --format='%H' 2>/dev/null)

LEDGER_ENABLED ?= true
ROCKSDB_ENABLED ?= true

###############################################################################
###                          Directories & Binaries                         ###
###############################################################################

ROOT ?= $(CURDIR)
GO ?= go
GOPATH ?= $(shell $(GO) env GOPATH 2>/dev/null)

BINDIR ?= $(if $(GOPATH),$(GOPATH)/bin,$(HOME)/go/bin)
BUILDDIR ?= $(ROOT)/build

CTM_BINARY := ctmd
LINUX_LIBDIR ?= /usr/local/lib

DISTDIR ?= $(BUILDDIR)/dist
DIST_BINDIR := $(DISTDIR)/bin
DIST_LIBDIR := $(DISTDIR)/lib

###############################################################################
###                              Repo Info                                  ###
###############################################################################

HTTPS_GIT := https://github.com/c8ntinuum/continuum.git
DOCKER := $(shell command -v docker 2>/dev/null)
ifndef DOCKER
  $(warning docker not found in PATH; docker-based targets (if any) will not work)
endif

export GO111MODULE = on

###############################################################################
###                            Submodule Settings                           ###
###############################################################################

EVMD_DIR      := evmd
EVMD_MAIN_PKG := ./cmd/evmd
