.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build core and plugins
	buildcli buildplugins

.PHONY: build-plugins
build-plugins: ## Build plugins
	go build -o ~/.scanio/plugins/gitlab ./plugins/gitlab/
	go build -o ~/.scanio/plugins/github ./plugins/github/
	go build -o ~/.scanio/plugins/bitbucket ./plugins/bitbucket/
	go build -o ~/.scanio/plugins/semgrep ./plugins/semgrep/
	go build -o ~/.scanio/plugins/bandit ./plugins/bandit/

.PHONY: build-cli
build-cli: ## Build scanio core
	go build -o ~/.local/bin/scanio .

.PHONE: docker
docker: ## Build docker image
	docker build -f dockerfiles/Dockerfile -t scanio .

.PHONE: helm-clean
helm-clean: ## Uninstall all helm releases
	helm ls --all --short | xargs -L1 helm delete
