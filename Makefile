.POSIX:

GO := go
GOLINT := golangci-lint

NX := ./cmd/nx
SOURCE := ./lib/source

BIN := bin

.PHONY: all
all: build

.PHONY: build
build: fmt source nx

source:
	$(GO) build $(SOURCE)

nx:
	$(GO) build -o $(BIN)/$@ $(NX)

.PHONY: fmt tidy clean
fmt:
	$(GO) fmt ./...

tidy:
	$(GO) mod tidy

clean:
	rm -fr bin/*

.PHONY: test lint
test:
	$(GO) test ./... -race $(ARGS)

lint:
	$(GOLINT) run
