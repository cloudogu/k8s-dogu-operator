$(GOPATH)/bin/go-junit-report:
	@echo "Download test dependencies with GO111MODULE=${GOMODULES}"
	@GO111MODULE=${GOMODULES} $(GO_CALL) get -u github.com/jstemmer/go-junit-report
