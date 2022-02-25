# Set these to the desired values
ARTIFACT_ID=k8s-dogu-operator
VERSION=0.0.1

# Image URL to use all building/pushing image targets
IMG ?= cloudogu/${ARTIFACT_ID}:${VERSION}

MAKEFILES_VERSION=4.7.1

.DEFAULT_GOAL:=

include build/make/variables.mk
include build/make/self-update.mk
include build/make/info.mk
include build/make/dependencies-gomod.mk
include build/make/build.mk
include build/make/test-common.mk
include build/make/test-integration.mk
include build/make/test-unit.mk
include build/make/static-analysis.mk
include build/make/clean.mk
include build/make/digital-signature.mk

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.23
K8S_INTEGRATION_TEST_DIR=${TARGET_DIR}/k8s-integration-test
K8S_UTILITY_BIN_PATH=$(WORKDIR)/.bin
K8S_RESOURCE_YAML=$(TARGET_DIR)/${ARTIFACT_ID}_${VERSION}.yaml

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development (without go container)

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	@echo "Generate manifests..."
	@$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	@echo "Auto-generate deepcopy functions..."
	@$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: vet
vet: $(STATIC_ANALYSIS_DIR) ## Run go vet against code.
	@go vet ./... | tee ${STATIC_ANALYSIS_DIR}/report-govet.out

$(K8S_INTEGRATION_TEST_DIR):
	@mkdir -p $@

.PHONY: k8s-integration-test
k8s-integration-test: $(K8S_INTEGRATION_TEST_DIR) manifests generate vet envtest ## Run k8s integration tests.
	@echo "Running k8s integration tests..."
	@KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test -tags=k8s_integration ./... -coverprofile ${K8S_INTEGRATION_TEST_DIR}/report-k8s-integration.out

##@ Build

.PHONY: build
build: generate vet ## Build controller binary.
	go build -o ${TARGET_DIR}/${ARTIFACT_ID} main.go

.PHONY: run
run: manifests generate vet ## Run a controller from your host.
	go run ./main.go

##@ Release

# todo[jsprey] wip
.PHONY: controller-release
controller-release:
	build/make/release.sh controller-tool

##@ Docker-related functions

.PHONY: docker-build
docker-build: ## Build docker image with the controller.
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the controller.
	docker push ${IMG}

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

${K8S_RESOURCE_YAML}: ${TARGET_DIR} manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default > ${K8S_RESOURCE_YAML}

.PHONY: k8s-generate
k8s-generate: ${K8S_RESOURCE_YAML} ## Create required k8s resources in ./dist/...
	@echo "Generating new kubernetes resources..."

.PHONY: k8s-deploy
k8s-deploy: k8s-generate ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cat ${K8S_RESOURCE_YAML} | kubectl apply -f -

.PHONY: k8s-undeploy
k8s-undeploy: k8s-generate ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	cat ${K8S_RESOURCE_YAML} | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

##@ Download Kubernetes Utility Tools

CONTROLLER_GEN = $(K8S_UTILITY_BIN_PATH)/controller-gen
.PHONY: controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.8.0)

KUSTOMIZE = $(K8S_UTILITY_BIN_PATH)/kustomize
.PHONY: kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

ENVTEST = $(K8S_UTILITY_BIN_PATH)/setup-envtest
.PHONY: envtest
envtest: ## Download envtest-setup locally if necessary.
	$(call go-get-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)

# go-get-tool will 'go get' any package $2 and install it to $1.
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(K8S_UTILITY_BIN_PATH) go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
