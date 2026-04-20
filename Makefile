.PHONY: build test vet cross-build clean wrappers

# Default target platform: current host.
# Override with `make build GOOS=linux GOARCH=arm64` etc.
GOOS  ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

BIN_DIR := lib/exec
PERFMONGER_BIN := $(BIN_DIR)/perfmonger_$(GOOS)_$(GOARCH)
CORE_BIN := $(BIN_DIR)/perfmonger-core_$(GOOS)_$(GOARCH)

# Compatibility wrapper symlinks (legacy names that point to perfmonger-core).
CORE_SUBCMDS := recorder player viewer summarizer plot-formatter

build: $(PERFMONGER_BIN) $(CORE_BIN) wrappers

$(PERFMONGER_BIN):
	mkdir -p $(BIN_DIR)
	cd core/cmd/perfmonger && GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o ../../../$(PERFMONGER_BIN) .

$(CORE_BIN):
	mkdir -p $(BIN_DIR)
	cd core/cmd/perfmonger-core && GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o ../../../$(CORE_BIN) perfmonger-core.go

wrappers: $(CORE_BIN)
	@for sub in $(CORE_SUBCMDS); do \
		target="$(BIN_DIR)/perfmonger-$${sub}_$(GOOS)_$(GOARCH)"; \
		ln -sf "perfmonger-core_$(GOOS)_$(GOARCH)" "$$target"; \
	done

test:
	cd core/internal/perfmonger && go test -v -cover
	uv sync && uv run pytest -v

vet:
	cd core/internal/perfmonger && go vet perfmonger_linux.go $$(ls *.go | grep -v perfmonger_)

cross-build:
	$(MAKE) build GOOS=linux GOARCH=amd64
	$(MAKE) build GOOS=linux GOARCH=arm64

clean:
	rm -f $(BIN_DIR)/perfmonger_* $(BIN_DIR)/perfmonger-*
