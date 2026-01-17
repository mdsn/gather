.POSIX:

GO := go
GOLINT := golangci-lint

GATHER := ./cmd/gather
SOURCE := ./lib/source

BIN := bin

.PHONY: all
all: build

.PHONY: build
build: fmt source gather

source:
	$(GO) build $(SOURCE)

gather:
	$(GO) build -o $(BIN)/$@ $(GATHER)

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
