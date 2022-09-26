
.PHONY: build
build: buildcli buildplugins

.PHONY: buildplugins
buildplugins:
	go build -o ~/.scanio/plugins/github ./plugins/github/ && \
	go build -o ~/.scanio/plugins/semgrep ./plugins/semgrep/

.PHONY: buildcli
buildcli:
	go build -o ~/.local/bin/scanio .

.PHONY: runprojects
runprojects:
	go run main.go fetch --vcs github --projects github.com/gitsight/go-vcsurl

.PHONY: runorg
runorg:
	go run main.go fetch --vcs github --org bookingcom -j 5

.PHONY: clean
clean:
	rm -rf ~/.scanio/projects/*

.PHONE: runscan
runscan:
	go run main.go analyse --scanner semgrep --projects github.com/bookingcom/telegraf,github.com/bookingcom/carbonapi

.PHONE: docker
docker:
	docker build -f dockerfiles/Dockerfile -t scanio .
