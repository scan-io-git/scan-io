# Makefile for Custom Scanio Build

# Default variables
SCANIO_REPO ?= https://github.com/scan-io-git/scan-io.git
SCANIO_REPO_DIR ?= ./scan-io
SCANIO_REPO_BRANCH ?=
SCANIO_LOCAL_PATH ?=
SCANIO_REPO_DIR_ABS := $(abspath $(SCANIO_REPO_DIR))
IMAGE_NAME ?= scanio
APP_NAME ?= scanio
RULES_CONFIG ?= scanio_rules.yaml
SCANIO_CONFIG ?= config.yml
INSTALL_DEPS_SCRIPT ?= install-deps.sh
CLONED_CONFIG_PATH ?= $(SCANIO_REPO_DIR)/$(notdir $(SCANIO_CONFIG))
CLONED_RULES_PATH ?= $(SCANIO_REPO_DIR)/scripts/rules/$(notdir $(RULES_CONFIG))
VERSION ?= 1.0
TARGET_OS ?= linux
TARGET_ARCH ?= amd64
REGISTRY ?=
PLUGINS ?= github gitlab bitbucket semgrep bandit trufflehog

# Help target
.PHONY: help
help: ## Display this help message
	@echo "Usage: make <target> [options]"
	@echo "Options:"
	@echo "  SCANIO_REPO      - URL of the Scanio repo (default: https://github.com/scan-io-git/scan-io.git)"
	@echo "  IMAGE_NAME		  - Tag name of the image  (default: scanio)"
	@echo "  SCANIO_REPO_DIR  - Directory for cloning the Scanio repository  (default: ./scan-io)"
	@echo "  SCANIO_REPO_BRANCH - Branch, tag, or commit to checkout after clone (default: repo default)"
	@echo "  SCANIO_LOCAL_PATH  - Path to a local Scanio checkout. When set, copies it into SCANIO_REPO_DIR instead of cloning SCANIO_REPO (default: empty)"
	@echo "  RULES_CONFIG     - Path to the custom rule set config (default: ./scanio_rules.yaml)"
	@echo "  SCANIO_CONFIG     - Path to the custom Scanio core config (default: ./config.yml)"
	@echo "  CLONED_CONFIG_PATH     - Destination path for Scanio core config file in the cloned repo (default: <SCANIO_REPO_DIR>/<SCANIO_CONFIG>)"
	@echo "  CLONED_RULES_PATH     - Destination path for the rule set config file in the cloned repo (default: <SCANIO_REPO_DIR>/scripts/rules/<RULES_CONFIG>)"
	@echo "  VERSION          - Docker image version (default: 1.0)"
	@echo "  TARGET_OS        - Target OS (default: linux)"
	@echo "  TARGET_ARCH      - Target architecture (default: amd64)"
	@echo "  REGISTRY         - Docker registry (default: empty)"
	@echo "  APP_NAME         - In-image binary name and path root (default: scanio)"
	@echo "Example:"
	@echo "  make build SCANIO_REPO=<repo_url> RULES_CONFIG=<rules_path> VERSION=<version> TARGET_OS=<os> TARGET_ARCH=<arch> REGISTRY=<registry>"
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Clone the Scanio repository
.PHONY: clone-scanio-repo
clone-scanio-repo: ## Clone the Scanio repository (or copy from SCANIO_LOCAL_PATH if set)
	@if [ -d $(SCANIO_REPO_DIR) ]; then \
		echo "[Custom Makefile] Removing existing Scanio repository '$(SCANIO_REPO_DIR)' ..."; \
		rm -rf $(SCANIO_REPO_DIR); \
	fi
	@if [ -n "$(SCANIO_LOCAL_PATH)" ]; then \
		if [ ! -d "$(SCANIO_LOCAL_PATH)" ]; then \
			echo "Local path '$(SCANIO_LOCAL_PATH)' does not exist"; exit 1; \
		fi; \
		if [ ! -f "$(SCANIO_LOCAL_PATH)/Makefile" ]; then \
			echo "Local path '$(SCANIO_LOCAL_PATH)' does not look like a Scanio repository (no Makefile found)"; exit 1; \
		fi; \
		src="$(SCANIO_LOCAL_PATH)"; \
		dst="$(SCANIO_REPO_DIR_ABS)"; \
		exclude_arg=""; \
		case "$$dst" in \
			"$$src"/*) \
				rel="$${dst#$$src/}"; \
				exclude_arg="--exclude=/$$rel"; \
				;; \
		esac; \
		echo "[Custom Makefile] Copying local Scanio repo from '$(SCANIO_LOCAL_PATH)' to '$(SCANIO_REPO_DIR)'"; \
		rsync -a $$exclude_arg "$$src/" "$$dst/" || { echo "Failed to copy $(SCANIO_LOCAL_PATH)"; exit 1; }; \
	else \
		echo "[Custom Makefile] Cloning Scanio repository to '$(SCANIO_REPO_DIR)'"; \
		git clone $(SCANIO_REPO) $(SCANIO_REPO_DIR) || { echo "Failed to clone $(SCANIO_REPO)"; exit 1; }; \
		if [ -n "$(SCANIO_REPO_BRANCH)" ]; then \
			echo "[Custom Makefile] Checking out $(SCANIO_REPO_BRANCH)..."; \
			cd $(SCANIO_REPO_DIR) && git checkout $(SCANIO_REPO_BRANCH) || { echo "Failed to checkout $(SCANIO_REPO_BRANCH)"; exit 1; }; \
		fi; \
	fi

# Copy custom config to cloned Scanio repo
.PHONY: copy-config
copy-config: ## Copy the custom config file to the cloned Scanio repository
	@if [ -f "$(SCANIO_CONFIG)" ]; then \
		echo "[Custom Makefile] Copying config file from $(SCANIO_CONFIG) to $(CLONED_CONFIG_PATH)..."; \
		cp $(SCANIO_CONFIG) $(CLONED_CONFIG_PATH) || { echo "Failed to copy $(SCANIO_CONFIG)"; exit 1; }; \
	else \
		echo "[Custom Makefile] Config file $(SCANIO_CONFIG) not found! Skipping..."; \
	fi

# Copy custom rule set to cloned Scanio repo
.PHONY: copy-rules
copy-rules: ## Copy the custom rule set to the cloned Scanio repository
	@if [ -f "$(RULES_CONFIG)" ]; then \
		echo "[Custom Makefile] Copying custom rule set from $(RULES_CONFIG) to $(CLONED_RULES_PATH)..."; \
		mkdir -p $(dir $(CLONED_RULES_PATH)); \
		cp $(RULES_CONFIG) $(CLONED_RULES_PATH) || { echo "Failed to copy $(RULES_CONFIG)"; exit 1; }; \
	else \
		echo "[Custom Makefile] Custom rule set $(RULES_CONFIG) not found! Skipping..."; \
	fi

# Copy optional dependency overlay script into the cloned repo
.PHONY: copy-deps-script
copy-deps-script: ## Copy the optional dependency overlay script to the cloned Scanio repository
	@if [ -f "$(INSTALL_DEPS_SCRIPT)" ]; then \
		echo "[Custom Makefile] Copying dependency overlay from $(INSTALL_DEPS_SCRIPT) to $(SCANIO_REPO_DIR)/docker-context-files/install-deps.sh..."; \
		mkdir -p $(SCANIO_REPO_DIR)/docker-context-files; \
		cp $(INSTALL_DEPS_SCRIPT) $(SCANIO_REPO_DIR)/docker-context-files/install-deps.sh || { echo "Failed to copy $(INSTALL_DEPS_SCRIPT)"; exit 1; }; \
	else \
		echo "[Custom Makefile] Dependency overlay $(INSTALL_DEPS_SCRIPT) not found! Using repo default (no-op)..."; \
	fi

# Build rules
.PHONY: build-rules
build-rules: ## Build rules in the cloned Scanio repository
	@echo "[Custom Makefile] Building rule sets in $(SCANIO_REPO_DIR)..."
	@cd $(SCANIO_REPO_DIR) && make build-rules USE_VENV=true FORCE=true || { echo "Failed to build rules"; exit 1; }

# Build Docker image
.PHONY: build-docker
build-docker: ## Build Docker image in the cloned Scanio repo
	@echo "[Custom Makefile] Building Docker image for $(TARGET_OS)/$(TARGET_ARCH)..."
	@cd $(SCANIO_REPO_DIR) && make docker-build PLUGINS="$(PLUGINS)" TARGETOS=$(TARGET_OS) TARGETARCH=$(TARGET_ARCH) REGISTRY=$(REGISTRY) IMAGE_NAME=$(IMAGE_NAME) APP_NAME=$(APP_NAME) VERSION=$(VERSION) || { echo "Failed to build Docker image"; exit 1; }

# Push Docker image to registry
.PHONY: push-docker
push-docker: ## Push Docker image to the registry
	@echo "[Custom Makefile] Pushing Docker image to: $(REGISTRY)..."
	@cd $(SCANIO_REPO_DIR) && make docker-push REGISTRY=$(REGISTRY) IMAGE_NAME=$(IMAGE_NAME) VERSION=$(VERSION) || { echo "Failed to push Docker image"; exit 1; }

# Clean the repository and local files
.PHONY: clean
clean: ## Clean the cloned repository and generated files
	@echo "[Custom Makefile] Cleaning up '$(SCANIO_REPO_DIR)'..."
	@rm -rf $(SCANIO_REPO_DIR)

# Build process to call all steps in sequence
.PHONY: build
build: clone-scanio-repo copy-config copy-rules copy-deps-script build-rules build-docker push-docker  ## Run full orchestrator process to build, package, and push
	@echo "[Custom Makefile] Custom Scanio build process complete!"
