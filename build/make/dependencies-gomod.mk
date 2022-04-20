##@ Go mod dependency management

.PHONY: dependencies
dependencies: vendor ## Install dependencies using go mod

vendor: go.mod go.sum
	@echo "Installing dependencies using go modules..."
	${GO_CALL} mod vendor

# go-get-tool will 'go get' any package $2 and install it to $1.
define go-get-tool
	@[ -f $(1) ] || { \
		set -e ;\
		TMP_DIR=$$(mktemp -d) ;\
		cd $$TMP_DIR ;\
		go mod init tmp ;\
		echo "Downloading $(2)" ;\
		GOBIN=$(UTILITY_BIN_PATH) go install $(2) ;\
		rm -rf $$TMP_DIR ;\
	}
endef
