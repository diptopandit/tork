# ─── tork Makefile ────────────────────────────────────────────────────────────

MODULE   := github.com/diptopandit/tork
BUILD_DIR := build
DIST_DIR  := dist
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE     ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(DATE)

TUI_PKG  := ./cmd/tork
CLI_PKG  := ./cmd/tork-cli

TUI_BIN  := $(BUILD_DIR)/tork
CLI_BIN  := $(BUILD_DIR)/tork-cli

# Cross-compilation targets: OS_ARCH
PLATFORMS := \
	darwin_amd64 \
	darwin_arm64 \
	linux_amd64 \
	linux_arm64 \
	windows_amd64 \
	windows_arm64

.PHONY: all build tui cli clean test vet fmt lint install release release-all help

## all: build both binaries for the current platform (default)
all: build

## build: compile tork and tork-cli for the current platform
build: tui cli

## tui: build the TUI binary → build/tork
tui: $(TUI_BIN)

$(TUI_BIN): $(shell find . -name '*.go' -not -path './build/*' -not -path './dist/*')
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $@ $(TUI_PKG)

## cli: build the CLI binary → build/tork-cli
cli: $(CLI_BIN)

$(CLI_BIN): $(shell find . -name '*.go' -not -path './build/*' -not -path './dist/*')
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $@ $(CLI_PKG)

## clean: remove build and dist artifacts
clean:
	rm -rf $(BUILD_DIR) $(DIST_DIR)

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
	@for i in $$(seq 1 40); do \
		docker exec tork-mysql-test mysql -h localhost -ptork -e "SELECT 1" tork_test >/dev/null 2>&1 && break; \
		sleep 3; \
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

## version: print the resolved version string
version:
	@echo $(VERSION)

## release: build for a single platform (usage: make release PLATFORM=darwin_arm64)
release:
ifndef PLATFORM
	$(error PLATFORM is required, e.g. make release PLATFORM=darwin_arm64)
endif
	$(eval OS   := $(word 1,$(subst _, ,$(PLATFORM))))
	$(eval ARCH := $(word 2,$(subst _, ,$(PLATFORM))))
	$(eval EXT  := $(if $(filter windows,$(OS)),.exe,))
	$(eval OUT  := $(DIST_DIR)/tork_$(VERSION)_$(OS)_$(ARCH))
	@mkdir -p $(OUT)
	GOOS=$(OS) GOARCH=$(ARCH) CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(OUT)/tork$(EXT) $(TUI_PKG)
	GOOS=$(OS) GOARCH=$(ARCH) CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(OUT)/tork-cli$(EXT) $(CLI_PKG)
	@cp LICENSE $(OUT)/
	@cp README.md $(OUT)/
	@if [ "$(OS)" = "windows" ]; then \
		(cd $(DIST_DIR) && zip -qr tork_$(VERSION)_$(OS)_$(ARCH).zip tork_$(VERSION)_$(OS)_$(ARCH)); \
	else \
		tar -czf $(DIST_DIR)/tork_$(VERSION)_$(OS)_$(ARCH).tar.gz -C $(DIST_DIR) tork_$(VERSION)_$(OS)_$(ARCH); \
	fi
	@echo "→ $(DIST_DIR)/tork_$(VERSION)_$(OS)_$(ARCH)$(if $(filter windows,$(OS)),.zip,.tar.gz)"

## release-all: build release archives for all platforms
release-all:
	@rm -rf $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		echo "Building $$platform..."; \
		$(MAKE) --no-print-directory release PLATFORM=$$platform; \
	done
	@echo ""
	@echo "Release archives in $(DIST_DIR)/:"
	@ls -1 $(DIST_DIR)/*.tar.gz $(DIST_DIR)/*.zip 2>/dev/null

## checksums: generate SHA-256 checksums for all release archives
checksums:
	@cd $(DIST_DIR) && shasum -a 256 *.tar.gz *.zip 2>/dev/null > checksums.txt
	@echo "→ $(DIST_DIR)/checksums.txt"

## help: show this help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'
