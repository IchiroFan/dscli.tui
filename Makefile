# dscli.tui — Terminal UI frontend for dscli
#
# Targets:
#   build        Compile the binary
#   install      Install to $GOPATH/bin (or $PREFIX/bin)
#   test         Run tests with race detector and coverage
#   check-dscli  Verify the dscli executable is available
#   clean        Remove build artifacts
#   help         Show this help

# ─── Project metadata ──────────────────────────────────────────────────────────

BINARY   := dscli-tui
MODULE   := gitcode.com/dscli/dscli.tui
MAIN     := ./cmd/$(BINARY)

# ─── Version injection ─────────────────────────────────────────────────────────

GIT_TAG    := $(shell git describe --tags --always --dirty 2>/dev/null || echo "devel")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS   := -ldflags "-X main.version=$(GIT_TAG) -X main.commit=$(GIT_COMMIT) -X main.date=$(BUILD_TIME)"

# ─── Paths ─────────────────────────────────────────────────────────────────────

BUILD_DIR  := ./build
OUTPUT     := $(BUILD_DIR)/$(BINARY)

# Install destination: prefer $GOPATH/bin, fall back to $HOME/.local/bin
GOPATH     := $(shell go env GOPATH 2>/dev/null)
PREFIX     ?= $(or $(GOPATH),$(HOME)/.local)
INSTALL_DIR:= $(PREFIX)/bin
INSTALLED  := $(INSTALL_DIR)/$(BINARY)

# ─── Tooling ───────────────────────────────────────────────────────────────────

GO     := go
GOTEST := $(GO) test
GOVET  := $(GO) vet

# ─── Flags ─────────────────────────────────────────────────────────────────────

GOFLAGS   ?=
TAGS      ?=
TESTFLAGS ?= -race -count=1 -coverprofile=coverage.out -covermode=atomic
VETFLAGS  ?= ./...

ifneq ($(TAGS),)
	GOFLAGS += -tags $(TAGS)
endif

# ─── OS / Arch helpers ─────────────────────────────────────────────────────────

os   := $(shell go env GOOS)
arch := $(shell go env GOARCH)

# ==============================================================================
#  Targets
# ==============================================================================

all: build  ## Default: build

build: $(OUTPUT)  ## Compile the binary

$(OUTPUT): go.mod $(shell find . -name '*.go' -type f)
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $@ $(MAIN)
	@echo "  → built:  $@ ($(os)/$(arch))"

install: $(INSTALLED)  ## Install to $PREFIX/bin (default: $GOPATH/bin)

$(INSTALLED): $(OUTPUT)
	@mkdir -p $(INSTALL_DIR)
	cp $(OUTPUT) $@
	@echo "  → installed: $@"

test:  ## Run all tests with race detector and coverage
	$(GOTEST) $(TESTFLAGS) ./...
	@echo "  → tests passed ✓"

vet:  ## Run go vet on all packages
	$(GOVET) $(VETFLAGS) ./...
	@echo "  → vet passed ✓"

check: check-dscli  ## Alias for check-dscli

check-dscli:  ## Verify the dscli executable is available in $PATH
	@echo "  → checking dscli ..."
	@command -v dscli >/dev/null 2>&1 \
		&& { echo "  ✓ dscli found at $$(command -v dscli)"; } \
		|| { echo "  ✗ dscli not found in PATH"; echo; \
		     echo "    dscli.tui requires the dscli CLI tool."; \
		     echo "    Install it from: https://github.com/dscli/dscli"; \
		     echo; exit 1; }

clean:  ## Remove build artifacts
	rm -rf $(BUILD_DIR) coverage.out coverage.html
	@echo "  → cleaned ✓"

coverage: test  ## Run tests and open coverage report
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "  → coverage report: coverage.html"

fmt:  ## Format all Go source files
	$(GO) fmt ./...
	@echo "  → formatted ✓"

lint:  ## Run staticcheck if available
	@command -v staticcheck >/dev/null 2>&1 \
		&& { staticcheck ./...; } \
		|| { echo "  → staticcheck not installed, skipping"; }

help:  ## Show this help
	@echo "Usage: make <target>"
	@echo
	@echo "Targets:"
	@awk -F '##' '/^[a-zA-Z_-]+:.*##/ { \
		gsub(/^[ \t]+|[ \t]+$$/, "", $$1); \
		printf "  %-18s  %s\n", $$1, $$2; \
	}' $(MAKEFILE_LIST)
	@echo
	@echo "Variables:"
	@echo "  PREFIX=$(PREFIX)    (install destination, default: \$$GOPATH/bin)"
	@echo "  TAGS=                (build tags, optional)"
	@echo "  GOFLAGS=             (extra Go flags, optional)"

# ─── Phony ─────────────────────────────────────────────────────────────────────

.PHONY: all build install test vet check check-dscli clean coverage fmt lint help
