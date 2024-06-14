# Makefile for building Scanio core and plugins

# Define variables
VERSION := $(shell cat VERSION)
GO_VERSION := $(shell go version | awk '{print $$3}')
BUILD_TIME := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
CORE_BINARY ?= ~/.local/bin/scanio
PLUGINS_DIR ?= ~/.scanio/plugins

# Help target
.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# Build all targets
.PHONY: build
build: build-cli build-plugins ## Build core and plugins

# Build Scanio core
.PHONY: build-cli
build-cli: check-go-dependency ## Build Scanio core
	@echo "Building Scanio core..."
	pwd $(CORE_BINARY); \
	go build -ldflags="-X 'github.com/scan-io-git/scan-io/cmd/version.CoreVersion=$(VERSION)' \
	                   -X 'github.com/scan-io-git/scan-io/cmd/version.GolangVersion=$(GO_VERSION)' \
	                   -X 'github.com/scan-io-git/scan-io/cmd/version.BuildTime=$(BUILD_TIME)'" \
	   -o $(CORE_BINARY) .

# Build plugins
.PHONY: build-plugins
build-plugins: check-go-dependency clean-plugins prepare-plugins ## Build Scanio plugins
	@echo "Building Scanio plugis..."
	@for dir in plugins/*/ ; do \
	    plugin_name=$$(basename $$dir); \
	    version=$$(cat $$dir/VERSION); \
	    output_dir=$(PLUGINS_DIR)/$$plugin_name; \
	    LDFLAGS_PLUGINS="-X main.Version=$$version \
	                     -X main.GolangVersion=$(GO_VERSION) \
	                     -X main.BuildTime=$(BUILD_TIME)"; \
	    echo "Building plugin: $$plugin_name v$$version"; \
		echo "Writing to $$output_dir"; \
	    mkdir -p $$output_dir; \
	    go build -ldflags "$$LDFLAGS_PLUGINS" -o $$output_dir/$$plugin_name ./$$dir || { echo "Failed to build plugin: $$plugin_name"; exit 1; }; \
	    cp $$dir/VERSION $$output_dir/VERSION; \
	done

# Build docker image
.PHONY: docker
docker: check-docker-dependency ## Build docker image
	docker build -t scanio .

# Uninstall all helm releases
.PHONY: helm-clean
helm-clean: ## Uninstall all helm releases
	helm ls --all --short | xargs -L1 helm delete

# Prepare plugins directory
.PHONY: prepare-plugins
prepare-plugins: ## Prepare plugins directory
	@if [ ! -d $(PLUGINS_DIR) ]; then \
		mkdir -p $(PLUGINS_DIR); \
	fi

# Clean plugins directory
.PHONY: clean-plugins
clean-plugins: ## Clean plugins directory
	rm -rf $(PLUGINS_DIR)/*

# Check for required commands
.PHONY: check-go-dependency
check-go-dependency:
	@command -v go >/dev/null 2>&1 || { echo >&2 "Go is not installed. Aborting."; exit 1; }

.PHONY: check-docker-dependency
check-docker-dependency:
	@command -v docker >/dev/null 2>&1 || { echo >&2 "Docker is not installed. Aborting."; exit 1; }