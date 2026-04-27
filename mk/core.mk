###############################################################################
###                           Module & Versioning                           ###
###############################################################################

VERSION ?= $(shell echo $(shell git describe --tags --always 2>/dev/null) | sed 's/^v//' )
TMVERSION = $(shell $(GO) list -m github.com/cometbft/cometbft 2>/dev/null | sed 's:.* ::')
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
RELEASEDIR ?= $(ROOT)/dist

CTM_BINARY := ctmd
LINUX_LIBDIR ?= /usr/local/lib
DOCKER ?= docker

DISTDIR ?= $(BUILDDIR)/dist
DIST_BINDIR := $(DISTDIR)/bin
DIST_LIBDIR := $(DISTDIR)/lib

###############################################################################
###                           Goal-Specific Checks                          ###
###############################################################################

ACTIVE_GOALS := $(if $(strip $(MAKECMDGOALS)),$(MAKECMDGOALS),$(.DEFAULT_GOAL))
CHECK_BUILD_PREREQS_GOALS ?= all build dist verify-dist package-dist install install-system \
	 test vet print-go-env
NEEDS_BUILD_PREREQS := $(if $(strip $(filter $(CHECK_BUILD_PREREQS_GOALS),$(ACTIVE_GOALS))),true,false)

###############################################################################
###                              Repo Info                                  ###
###############################################################################

HTTPS_GIT := https://github.com/c8ntinuum/continuum.git

export GO111MODULE = on

###############################################################################
###                            Submodule Settings                           ###
###############################################################################

CTM_DIR       := evmd
CTM_MAIN_PKG  := ./cmd/evmd
