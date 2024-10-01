# Makefile for building Scanio core, plugins, and managing Docker images

# Default variables
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
VENV_DIR ?= .venv
REQUIREMENTS_FILE ?= scripts/rules/requirements.txt
RULES_SCRIPT ?= scripts/rules/rules.py
RULES_CONFIG ?= scripts/rules/scanio_rules.yaml
RULES_DIR ?= ./rules
USE_VENV ?= false

# Default image tag
IMAGE_TAG := $(if $(REGISTRY),$(REGISTRY)/)$(IMAGE_NAME)

# Load environment variables from .env file if present
-include .env

.PHONY: help
help: ## Display available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: build-cli build-plugins ## Build Scanio core and plugins

.PHONY: build-cli
build-cli: check-go-dependency ## Build Scanio core
	@echo "Building Scanio core..."
	go build -ldflags="-X 'github.com/scan-io-git/scan-io/cmd/version.CoreVersion=$(VERSION)' \
	                   -X 'github.com/scan-io-git/scan-io/cmd/version.GolangVersion=$(GO_VERSION)' \
	                   -X 'github.com/scan-io-git/scan-io/cmd/version.BuildTime=$(BUILD_TIME)'" \
	   -o $(CORE_BINARY) . || exit 1
	@echo "Scanio core built successfully!"

# Build plugins
.PHONY: build-plugins
build-plugins: check-go-dependency check-jq-dependency clean-plugins prepare-plugins ## Build Scanio plugins
	@echo "Building Scanio plugins..."
	@for dir in plugins/*/ ; do \
	    plugin_name=$$(basename $$dir); \
	    version=$$(jq -r '.version' $$dir/VERSION); \
	    plugin_type=$$(jq -r '.plugin_type' $$dir/VERSION); \
	    output_dir=$(PLUGINS_DIR)/$$plugin_name; \
	    LDFLAGS_PLUGINS="-X main.Version=$$version \
	                     -X main.GolangVersion=$(GO_VERSION) \
	                     -X main.BuildTime=$(BUILD_TIME)"; \
	    echo "Building plugin '$$plugin_name' (v$$version, type: $$plugin_type)"; \
	    mkdir -p $$output_dir || { echo "Failed to create plugin directory: $$output_dir"; exit 1; }; \
	    go build -ldflags "$$LDFLAGS_PLUGINS" -o $$output_dir/$$plugin_name ./$$dir || { echo "Failed to build plugin: $$plugin_name"; exit 1; }; \
	    cp $$dir/VERSION $$output_dir/VERSION || { echo "Failed to copy VERSION for plugin: $$plugin_name"; exit 1; }; \
	done
	@echo "All Scanio plugins built successfully!"

.PHONY: setup-python-env
setup-python-env: ## Set up Python virtual environment and install dependencies
	@echo "Setting up Python virtual environment in $(VENV_DIR)..."
	@if [ ! -d $(VENV_DIR) ]; then \
		python3 -m venv $(VENV_DIR); \
	fi
	@$(VENV_DIR)/bin/pip install --upgrade pip || { echo "Failed to upgrade pip. Exiting."; exit 1; }
	@$(VENV_DIR)/bin/pip install -r $(REQUIREMENTS_FILE) || { echo "Failed to install dependencies from requirements.txt. Exiting."; exit 1; }
	@echo "Python virtual environment setup complete."

.PHONY: build-rules
build-rules: ## Build custom rule sets using Python script
	@if [ "$(USE_VENV)" = "true" ]; then \
		$(MAKE) setup-python-env || exit 1; \
		$(MAKE) check-python-dependency || exit 1; \
		echo "Building custom rule sets with virtual environment..."; \
		python_bin="$(VENV_DIR)/bin/python3"; \
	else \
		$(MAKE) check-python-dependency || exit 1; \
		echo "Building custom rule sets with system Python..."; \
		python_bin="python3"; \
	fi; \
	$$python_bin $(RULES_SCRIPT) -r $(RULES_CONFIG) --rules-dir $(RULES_DIR) $(if $(FORCE),--force) $(shell printf ' -v%.0s' $(VERBOSE)) || exit 1
	@echo "Custom rule sets built successfully!"

.PHONY: docker
docker: check-docker-dependency ## Build local Docker image (no registry push)
	@echo "Building local Docker image Scanio for personal use..."
	docker build -t $(IMAGE_NAME) .

# make docker-build VERSION=1.2 TARGETOS=linux TARGETARCH=amd64 REGISTRY=artifactory.example.com/security-tools/scanio
.PHONY: docker-build
docker-build: check-docker-dependency ## Build Docker image for production
	@echo "Building Docker image for $(TARGET_OS)/$(TARGET_ARCH)..."
	docker build --build-arg TARGETOS=$(TARGET_OS) --build-arg TARGETARCH=$(TARGET_ARCH) --platform=$(TARGET_OS)/$(TARGET_ARCH) \
	-t $(IMAGE_TAG):$(VERSION) -t $(IMAGE_TAG):latest . || exit 1
	@echo "Docker image built successfully."

# make docker-push REGISTRY=artifactory.example.com/security-tools/scanio VERSION=1.2
.PHONY: docker-push
docker-push: ## Push Docker image to registry
	@echo "Pushing Docker image to $(REGISTRY)..."
	docker push $(IMAGE_TAG):$(VERSION) || exit 1
	docker push $(IMAGE_TAG):latest || exit 1
	@echo "Docker images pushed to $(REGISTRY)."

.PHONY: clean-docker-images
clean-docker-images: ## Clean local Docker images
	@echo "Removing Docker images..."
	docker rmi -f $(IMAGE_NAME):$(VERSION) $(IMAGE_NAME):latest || true

.PHONY: clean-python-env
clean-python-env: ## Remove Python virtual environment
	@echo "Cleaning Python virtual environment..."
	rm -rf $(VENV_DIR)

.PHONY: helm-clean
helm-clean: ## Uninstall all helm releases
	helm ls --all --short | xargs -L1 helm delete

.PHONY: clean-plugins
clean-plugins: ## Clean plugin directory
	@rm -rf $(PLUGINS_DIR)/*

.PHONY: prepare-plugins
prepare-plugins: ## Prepare plugin directory
	@if [ ! -d $(PLUGINS_DIR) ]; then \
		mkdir -p $(PLUGINS_DIR); \
	fi

.PHONY: clean
clean: clean-plugins clean-docker-images clean-python-env ## Clean all generated artifacts

# Check for required dependencies
.PHONY: check-go-dependency
check-go-dependency:
	@command -v go >/dev/null 2>&1 || { echo "Go is not installed. Aborting."; exit 1; }

.PHONY: check-docker-dependency
check-docker-dependency:
	@command -v docker >/dev/null 2>&1 || { echo "Docker is not installed. Aborting."; exit 1; }

.PHONY: check-jq-dependency
check-jq-dependency:
	@command -v jq >/dev/null 2>&1 || { echo "jq is not installed. Aborting."; exit 1; }

.PHONY: check-python-dependency
check-python-dependency:
	@if [ "$(USE_VENV)" = "true" ]; then \
		echo "Checking Python dependencies in virtual environment..."; \
		python_bin="$(VENV_DIR)/bin/python3"; \
	else \
		echo "Checking Python dependencies in system Python..."; \
		python_bin="python3"; \
	fi; \
	$$python_bin -c "import yaml" || { echo "PyYAML is not installed. Aborting."; exit 1; }; \
	$$python_bin -c "import colorama" || { echo "Colorama is not installed. Aborting."; exit 1; }; \
	$$python_bin -c "import tqdm" || { echo "tqdm is not installed. Aborting."; exit 1; }; \
	$$python_bin -c "import git" || { echo "GitPython is not installed. Aborting."; exit 1; }; \
	echo "All required Python dependencies are installed."

.PHONY: test
test: ## Run Go tests
	go test -v ./... && echo "All tests passed"
