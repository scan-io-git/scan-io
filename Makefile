
.PHONY: build
build: buildcli buildplugins

.PHONY: buildplugins
buildplugins:
	go build -o ~/.scanio/plugins/gitlab ./plugins/gitlab/ && \
	go build -o ~/.scanio/plugins/github ./plugins/github/ && \
	go build -o ~/.scanio/plugins/semgrep ./plugins/semgrep/

.PHONY: buildcli
buildcli:
	go build -o ~/.local/bin/scanio .

.PHONY: clean
clean:
	rm -rf ~/.scanio/projects/*

.PHONE: docker
docker:
	docker build -f dockerfiles/Dockerfile -t scanio .
