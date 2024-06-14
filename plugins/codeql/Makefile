# Makefile for Codeql plugin

PLUGIN_NAME := codeql
VERSION := $(shell cat VERSION)
GO_VERSION := $(shell go version | awk '{print $$3}')
BUILD_TIME := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
PLUGINS_DIR ?= ~/.scanio/plugins

.PHONY: build
build: check-go-dependency ## Build plugin
	mkdir -p $(PLUGINS_DIR) && go build -ldflags="-X main.Version=$(VERSION) \
	                     -X main.GolangVersion=$(GO_VERSION) \
	                     -X main.BuildTime=$(BUILD_TIME)" \
						-o $(PLUGINS_DIR)/${PLUGIN_NAME}/$(PLUGIN_NAME)

# Check for required commands
.PHONY: check-go-dependency
check-go-dependency:
	@command -v go >/dev/null 2>&1 || { echo >&2 "Go is not installed. Aborting."; exit 1; }
