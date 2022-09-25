.PHONY: buildplugins
buildplugins:
	go build -o ~/.scanio/plugins/github ./plugins/github/

.PHONY: runprojects
runprojects:
	go run main.go fetch --vcs github --projects github.com/gitsight/go-vcsurl

.PHONY: runorg
runorg:
	go run main.go fetch --vcs github --org bookingcom
