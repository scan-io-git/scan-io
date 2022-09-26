.PHONY: buildplugins
buildplugins:
	go build -o ~/.scanio/plugins/github ./plugins/github/ && \
	go build -o ~/.scanio/plugins/semgrep ./plugins/semgrep/

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
	go run main.go analyse --scanner semgrep --project github.com/bookingcom/telegraf
