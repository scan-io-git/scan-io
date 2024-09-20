# Makefile for building Scanio core and plugins
# Define variables
VERSION := $(shell jq -r '.version' VERSION || echo "dev")
GO_VERSION := $(shell go version | awk '{print $$3}')
BUILD_TIME := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
CORE_BINARY ?= ~/.local/bin/scanio
PLUGINS_DIR ?= ~/.scanio/plugins
REGISTRY ?= 
IMAGE_NAME ?= scanio
TARGET_OS ?= linux
TARGET_ARCH ?= amd64
PLATFORM ?= linux/amd64

# Python script to build rule sets
RULES_SCRIPT ?= scripts/rules/rules.py
RULES_CONFIG ?= scripts/rules/scanio_rules.yaml  # Default rules config
RULES_DIR ?= ./rules  # Default output directory for rules

# Default image tag
IMAGE_TAG := $(if $(REGISTRY),$(REGISTRY)/)$(IMAGE_NAME)

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
build-plugins: check-go-dependency check-jq-dependency clean-plugins prepare-plugins ## Build Scanio plugins
	@echo "Building Scanio plugis..."
	@for dir in plugins/*/ ; do \
	    plugin_name=$$(basename $$dir); \
	    version=$$(jq -r '.version' $$dir/VERSION); \
	    plugin_type=$$(jq -r '.plugin_type' $$dir/VERSION); \
	    output_dir=$(PLUGINS_DIR)/$$plugin_name; \
	    LDFLAGS_PLUGINS="-X main.Version=$$version \
	                     -X main.GolangVersion=$(GO_VERSION) \
	                     -X main.BuildTime=$(BUILD_TIME)"; \
	    echo "Building plugin name: '$$plugin_name', version: 'v$$version', type: '$$plugin_type' "; \
		echo "Writing to $$output_dir"; \
	    mkdir -p $$output_dir; \
	    go build -ldflags "$$LDFLAGS_PLUGINS" -o $$output_dir/$$plugin_name ./$$dir || { echo "Failed to build plugin: $$plugin_name"; exit 1; }; \
	    cp $$dir/VERSION $$output_dir/VERSION; \
	done

# Build custom rule sets using the Python script
# make build-rules RULES_CONFIG=path/to/scanio_rules.yaml RULES_DIR=path/to/output/directory FORCE=true VERBOSE=1
.PHONY: build-rules
build-rules: check-python-dependency ## Build custom rule sets from the YAML configuration
	@echo "Building custom rule sets using the script at $(RULES_SCRIPT)..."
	python3 $(RULES_SCRIPT) -r $(RULES_CONFIG) --rules-dir $(RULES_DIR) $(if $(FORCE),--force) $(shell printf ' -v%.0s' $(VERBOSE))


# Local Docker build for personal use
.PHONY: docker
docker: check-docker-dependency ## Build local Docker image (no registry push)
	@echo "Building local Docker image Scanio for personal use..."
	docker build -t $(IMAGE_NAME) .

# Production Scanio docker build with arguments
# make docker-build VERSION=1.2 TARGETOS=linux TARGETARCH=amd64 REGISTRY=artifactory.example.com/security-tools/scanio
.PHONY: docker-build
docker-build: check-docker-dependency ## Build production Docker image with custom arguments
	@echo "Building Docker image for $(TARGETOS)/$(TARGETARCH)..."
	docker build --build-arg TARGETOS=$(TARGET_OS) --build-arg TARGETARCH=$(TARGET_ARCH) --platform=$(TARGET_OS)/$(TARGET_ARCH) \
	-t $(IMAGE_TAG):$(VERSION) \
	-t $(IMAGE_TAG):latest .

# Push production Scanio docker image to registry
# make docker-push REGISTRY=artifactory.example.com/security-tools/scanio VERSION=1.2
.PHONY: docker-push
docker-push: ## Push docker image to the registry
	@echo "Pushing Docker image to $(REGISTRY)..."
	docker push $(IMAGE_TAG):$(VERSION)
	docker push $(IMAGE_TAG):latest

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

.PHONY: check-jq-dependency
check-jq-dependency:
	@command -v jq >/dev/null 2>&1 || { echo >&2 "jq is not installed. Aborting."; exit 1; }

.PHONY: check-python-dependency
check-python-dependency:
	@command -v python3 >/dev/null 2>&1 || { echo >&2 "Python 3 is not installed. Aborting."; exit 1; }
	@python3 -c "import yaml" >/dev/null 2>&1 || { echo >&2 "PyYAML is not installed. Aborting."; exit 1; }
	@python3 -c "import colorama" >/dev/null 2>&1 || { echo >&2 "Colorama is not installed. Aborting."; exit 1; }
	@python3 -c "import tqdm" >/dev/null 2>&1 || { echo >&2 "tqdm is not installed. Aborting."; exit 1; }
	@python3 -c "import git" >/dev/null 2>&1 || { echo >&2 "tqdm is not installed. Aborting."; exit 1; }

.PHONY: test
test: ## Run tests
	go test -v ./... && echo "All tests passed"
