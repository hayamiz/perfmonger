.PHONY: build test vet cross-build clean

# Default target platform: current host.
# Override with `make build GOOS=linux GOARCH=arm64` etc.
GOOS  ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

BIN_DIR := lib/exec
PERFMONGER_BIN := $(BIN_DIR)/perfmonger_$(GOOS)_$(GOARCH)

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
