.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: build-cli build-plugins ## Build core and plugins

.PHONY: build-plugins
build-plugins: ## Build plugins
	go build -o ~/.scanio/plugins/gitlab ./plugins/gitlab/
	go build -o ~/.scanio/plugins/github ./plugins/github/
	go build -o ~/.scanio/plugins/bitbucket ./plugins/bitbucket/
	go build -o ~/.scanio/plugins/semgrep ./plugins/semgrep/
	go build -o ~/.scanio/plugins/bandit ./plugins/bandit/
	go build -o ~/.scanio/plugins/trufflehog ./plugins/trufflehog/
	go build -o ~/.scanio/plugins/trufflehog3 ./plugins/trufflehog3/
	go build -o ~/.scanio/plugins/codeql ./plugins/codeql/

.PHONY: build-cli
build-cli: ## Build scanio core
	go build -o ~/.local/bin/scanio .

.PHONE: docker
docker: ## Build docker image
	docker build -t scanio .

.PHONE: helm-clean
helm-clean: ## Uninstall all helm releases
	helm ls --all --short | xargs -L1 helm delete
