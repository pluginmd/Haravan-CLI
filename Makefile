# haravan-cli build and release tasks.

BINARY   := haravan-cli
MODULE   := github.com/pluginmd/haravan-cli
VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT   := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE     := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  := -s -w \
            -X $(MODULE)/internal/build.Version=$(VERSION) \
            -X $(MODULE)/internal/build.Commit=$(COMMIT) \
            -X $(MODULE)/internal/build.Date=$(DATE)
PREFIX   ?= /usr/local

.PHONY: all build install uninstall clean test vet lint tidy run release-snapshot

all: build

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) .

install: build
	install -d $(PREFIX)/bin
	install -m 0755 $(BINARY) $(PREFIX)/bin/$(BINARY)
	@echo "installed: $(PREFIX)/bin/$(BINARY) ($(VERSION))"

uninstall:
	rm -f $(PREFIX)/bin/$(BINARY)

clean:
	rm -f $(BINARY) $(BINARY).exe coverage.out
	rm -rf dist/

vet:
	go vet ./...

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed"; exit 1; }
	golangci-lint run ./...

test:
	go test -race -count=1 -coverprofile=coverage.out ./...

tidy:
	go mod tidy

run: build
	./$(BINARY) $(ARGS)

release-snapshot:
	@command -v goreleaser >/dev/null 2>&1 || { echo "goreleaser not installed"; exit 1; }
	goreleaser release --snapshot --clean
