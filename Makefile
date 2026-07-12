.PHONY: build test vet cross-build clean tools release-check

# Default target platform: current host.
# Override with `make build GOOS=linux GOARCH=arm64` etc.
GOOS  ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

BIN_DIR := lib/exec
PERFMONGER_BIN := $(BIN_DIR)/perfmonger_$(GOOS)_$(GOARCH)

# GoReleaser is a maintainer-only pre-flight tool (not needed to build or test
# perfmonger). Pin it to a specific v2 release that stays within the `~> v2`
# range the release workflow (.github/workflows/release.yml) uses, so local
# runs match CI. Bump this single line to upgrade.
GORELEASER_VERSION ?= v2.17.0

# Resolve the install location `go install` uses: GOBIN when set (e.g. under
# 00_LOAD_GO_DEVENV.sh, which points it at the repo-local godevenv tree),
# otherwise GOPATH/bin. This keeps `make tools` and `make release-check`
# working both with and without the dev env loaded.
GOBIN_DIR := $(shell go env GOBIN)
ifeq ($(GOBIN_DIR),)
GOBIN_DIR := $(shell go env GOPATH)/bin
endif
GORELEASER := $(GOBIN_DIR)/goreleaser

build: $(PERFMONGER_BIN)

$(PERFMONGER_BIN):
	mkdir -p $(BIN_DIR)
	cd core/cmd/perfmonger && GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o ../../../$(PERFMONGER_BIN) .

test:
	cd core/internal/perfmonger && go test -v -cover
	uv sync && uv run pytest -v

vet:
	cd core/internal/perfmonger && go vet perfmonger_linux.go $$(ls *.go | grep -v perfmonger_)

cross-build:
	$(MAKE) build GOOS=linux GOARCH=amd64
	$(MAKE) build GOOS=linux GOARCH=arm64

clean:
	rm -f $(BIN_DIR)/perfmonger_* $(BIN_DIR)/perfmonger-* core/cmd/perfmonger/perfmonger

# Install the pinned GoReleaser into GOBIN. Requires a Go toolchain; with the
# default GOTOOLCHAIN=auto an older host Go will fetch the toolchain GoReleaser
# needs. Load 00_LOAD_GO_DEVENV.sh first to install into the repo-local tree.
tools:
	go install github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION)

# Maintainer pre-flight before cutting a release: validate .goreleaser.yaml and
# do a full local build without publishing (artifacts land in dist/).
release-check: tools
	$(GORELEASER) check
	$(GORELEASER) release --snapshot --clean
