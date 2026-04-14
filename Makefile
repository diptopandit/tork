# ─── tork Makefile ────────────────────────────────────────────────────────────

MODULE   := github.com/diptopandit/tork
BUILD_DIR := build
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE     ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

TUI_PKG  := ./cmd/tork
CLI_PKG  := ./cmd/tork-cli

TUI_BIN  := $(BUILD_DIR)/tork
CLI_BIN  := $(BUILD_DIR)/tork-cli

.PHONY: all build tui cli clean test vet fmt lint install help

## all: build both binaries (default)
all: build

## build: compile tork (TUI) and tork-cli
build: tui cli

## tui: build the TUI binary → build/tork
tui: $(TUI_BIN)

$(TUI_BIN): $(shell find . -name '*.go' -not -path './build/*')
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $@ $(TUI_PKG)

## cli: build the CLI binary → build/tork-cli
cli: $(CLI_BIN)

$(CLI_BIN): $(shell find . -name '*.go' -not -path './build/*')
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $@ $(CLI_PKG)

## clean: remove build artifacts
clean:
	rm -rf $(BUILD_DIR)

## test: run all tests
test:
	go test ./...

## test-integration: run MySQL integration tests (requires Docker)
test-integration:
	@docker rm -f tork-mysql-test 2>/dev/null || true
	docker run -d --name tork-mysql-test \
		-e MYSQL_ROOT_PASSWORD=tork \
		-e MYSQL_DATABASE=tork_test \
		-p 3306:3306 \
		--tmpfs /var/lib/mysql \
		mysql:8.0
	@echo "Waiting for MySQL to be ready..."
	@for i in $$(seq 1 30); do \
		docker exec tork-mysql-test mysqladmin ping -h localhost -ptork --silent 2>/dev/null && break; \
		sleep 2; \
	done
	TORK_MYSQL_DSN="root:tork@tcp(127.0.0.1:3306)/tork_test" go test -v ./... || (docker rm -f tork-mysql-test && exit 1)
	docker rm -f tork-mysql-test

## vet: run static analysis
vet:
	go vet ./...

## fmt: format all Go files
fmt:
	gofmt -s -w .

## lint: run golangci-lint (must be installed)
lint:
	golangci-lint run ./...

## install: install both binaries to $GOPATH/bin
install:
	go install -ldflags "$(LDFLAGS)" $(TUI_PKG)
	go install -ldflags "$(LDFLAGS)" $(CLI_PKG)

## help: show this help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'
