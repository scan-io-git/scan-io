# Makefile for Custom Scanio Build

# Default variables
SCANIO_REPO ?= https://github.com/scan-io-git/scan-io.git  # Scanio repository URL
SCANIO_REPO_DIR ?= ./scan-io                             # Local directory for the cloned repository
RULES_CONFIG ?= scanio_rules.yaml                        # Path to the custom rule set
CLONED_RULES_PATH ?= $(SCANIO_REPO_DIR)/scripts/rules/$(notdir $(RULES_CONFIG)) # Where to copy rule set in the cloned repo
VERSION ?= 1.0                                           # Docker image version
TARGET_OS ?= linux                                       # Target OS
TARGET_ARCH ?= amd64                                     # Target architecture
REGISTRY ?=                                              # Optional Docker registry URL

# Help target
.PHONY: help
help: ## Display this help message
	@echo "Usage: make <target> [options]"
	@echo "Options:"
	@echo "  SCANIO_REPO      - URL of the Scanio repo (default: https://github.com/scan-io-git/scan-io.git)"
	@echo "  RULES_CONFIG     - Path to the custom rule set config (default: ./scanio_rules.yaml)"
	@echo "  VERSION          - Docker image version (default: 1.0)"
	@echo "  TARGET_OS        - Target OS (default: linux)"
	@echo "  TARGET_ARCH      - Target architecture (default: amd64)"
	@echo "  REGISTRY         - Docker registry (default: empty)"
	@echo "Example:"
	@echo "  make build SCANIO_REPO=<repo_url> RULES_CONFIG=<rules_path> VERSION=<version> TARGET_OS=<os> TARGET_ARCH=<arch> REGISTRY=<registry>"
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'


# Clone the Scanio repository
.PHONY: clone-scanio-repo
clone-scanio-repo: ## Clone the Scanio repository
	@if [ -d $(SCANIO_REPO_DIR) ]; then \
		echo "Removing existing Scanio repository..."; \
		rm -rf $(SCANIO_REPO_DIR); \
	fi
	@git clone $(SCANIO_REPO) $(SCANIO_REPO_DIR) || { echo "Failed to clone $(SCANIO_REPO)"; exit 1; }


# Copy custom config to cloned Scanio repo
.PHONY: copy-config
copy-config: ## Copy the custom config file to the cloned Scanio repository
	@if [ -f "$(SCANIO_CONFIG)" ]; then \
		echo "Copying config file from $(SCANIO_CONFIG) to $(CLONED_CONFIG_PATH)..."; \
		cp $(SCANIO_CONFIG) $(CLONED_CONFIG_PATH) || { echo "Failed to copy $(SCANIO_CONFIG)"; exit 1; }; \
	else \
		echo "Config file $(SCANIO_CONFIG) not found! Skipping..."; \
	fi

# Copy custom rule set to cloned Scanio repo
.PHONY: copy-rules
copy-rules: ## Copy the custom rule set to the cloned Scanio repository
	@if [ -f "$(RULES_CONFIG)" ]; then \
		echo "Copying custom rule set from $(RULES_CONFIG) to $(CLONED_RULES_PATH)..."; \
		mkdir -p $(dir $(CLONED_RULES_PATH)); \
		cp $(RULES_CONFIG) $(CLONED_RULES_PATH) || { echo "Failed to copy $(RULES_CONFIG)"; exit 1; }; \
	else \
		echo "Custom rule set $(RULES_CONFIG) not found! Skipping..."; \
	fi

# Build rules
.PHONY: build-rules
build-rules: ## Build rules in the cloned Scanio repository
	@echo "Building rule sets in $(SCANIO_REPO_DIR)..."
	@cd $(SCANIO_REPO_DIR) && make build-rules USE_VENV=true FORCE=true || { echo "Failed to build rules"; exit 1; }

# Build Docker image
.PHONY: build-docker
build-docker: ## Build Docker image in the cloned Scanio repo
	@echo "Building Docker image for $(TARGET_OS)/$(TARGET_ARCH)..."
	@cd $(SCANIO_REPO_DIR) && make docker-build VERSION=$(VERSION) TARGETOS=$(TARGET_OS) TARGETARCH=$(TARGET_ARCH) REGISTRY=$(REGISTRY) || { echo "Failed to build Docker image"; exit 1; }

# Push Docker image to registry
.PHONY: push-docker
push-docker: ## Push Docker image to the registry
	@echo "Pushing Docker image to registry: $(REGISTRY)..."
	@cd $(SCANIO_REPO_DIR) && make docker-push VERSION=$(VERSION) REGISTRY=$(REGISTRY) || { echo "Failed to push Docker image"; exit 1; }

# Clean the repository and local files
.PHONY: clean
clean: ## Clean the cloned repository and generated files
	@echo "Cleaning up..."
	@rm -rf $(SCANIO_REPO_DIR)

# Build process to call all steps in sequence
.PHONY: build
build: clone-scanio-repo copy-config copy-rules build-rules build-docker push-docker  ## Run full orchestrator process to build, package, and push
	@echo "Custom Scanio build process complete!"
