# Formattington — Go build and test helpers.
#
# Targets:
#   make build   Build the formattington binary into ./bin
#   make test    Run the test suite
#   make check   Vet, format check, and test
#   make clean   Remove build output

BIN_DIR := bin
BINARY := $(BIN_DIR)/formattington
PKG := ./cmd/formattington

GO ?= go

.PHONY: all build test vet fmt fmt-check check clean

all: build

build:
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BINARY) $(PKG)

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

fmt:
	$(GO) fmt ./...

fmt-check:
	@out="$$($(GO) fmt ./...)"; \
	if [ -n "$$out" ]; then \
		echo "gofmt needed for:"; \
		echo "$$out"; \
		exit 1; \
	fi

check: vet fmt-check test

clean:
	rm -rf $(BIN_DIR)
