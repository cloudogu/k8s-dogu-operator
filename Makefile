# Set these to the desired values
ARTIFACT_ID=k8s-dogu-operator
VERSION=0.1.0

GOTAG?=1.17.7
MAKEFILES_VERSION=5.0.0

# Image URL to use all building/pushing image targets
IMAGE=cloudogu/${ARTIFACT_ID}:${VERSION}

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.23
K8S_INTEGRATION_TEST_DIR=${TARGET_DIR}/k8s-integration-test
K8S_RESOURCE_YAML=${TARGET_DIR}/${ARTIFACT_ID}_${VERSION}.yaml

# make sure to create a statically linked binary otherwise it may quit with
# "exec user process caused: no such file or directory"
GO_BUILD_FLAGS=-mod=vendor -a -tags netgo,osusergo $(LDFLAGS) -o $(BINARY)
# remove DWARF symbol table and strip other symbols to shave ~13 MB from binary
ADDITIONAL_LDFLAGS=-extldflags -static -w -s

.DEFAULT_GOAL:=help

include build/make/variables.mk

ADDITIONAL_CLEAN=dist-clean
PRE_COMPILE=generate vet

include build/make/self-update.mk
include build/make/dependencies-gomod.mk
include build/make/build.mk
include build/make/test-common.mk
include build/make/test-integration.mk
include build/make/test-unit.mk
include build/make/static-analysis.mk
include build/make/clean.mk
include build/make/digital-signature.mk


# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

##@ EcoSystem

.PHONY: build
build: docker-build image-import k8s-apply ## Builds a new version of the setup and deploys it into the K8s-EcoSystem.

##@ Development (without go container)

${STATIC_ANALYSIS_DIR}/report-govet.out: ${SRC} $(STATIC_ANALYSIS_DIR)
	@go vet ./... | tee $@

.PHONY: vet
vet: ${STATIC_ANALYSIS_DIR}/report-govet.out ## Run go vet against code.

##@ Kubernetes Controller

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	@echo "Generate manifests..."
	@$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	@echo "Auto-generate deepcopy functions..."
	@$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

$(K8S_INTEGRATION_TEST_DIR):
	@mkdir -p $@

.PHONY: k8s-integration-test
k8s-integration-test: $(K8S_INTEGRATION_TEST_DIR) manifests generate vet envtest ## Run k8s integration tests.
	@echo "Running k8s integration tests..."
	@KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test -tags=k8s_integration ./... -coverprofile ${K8S_INTEGRATION_TEST_DIR}/report-k8s-integration.out

##@ Build

.PHONY: build-controller
build-controller: ${SRC} compile ## Builds the controller Go binary.

.PHONY: run
run: manifests generate vet ## Run a controller from your host.
	go run ./main.go

##@ Release

.PHONY: controller-release
controller-release: ## Interactively starts the release workflow.
	@echo "Starting git flow release..."
	@build/make/release.sh controller-tool

##@ Docker

.PHONY: docker-build
docker-build: ${SRC} ## Builds the docker image of the k8s-ces-setup `cloudogu/k8s-ces-setup:version`.
	@echo "Building docker image of dogu..."
	DOCKER_BUILDKIT=1 docker build . -t ${IMAGE}

${K8S_CLUSTER_ROOT}/image.tar: check-k8s-cluster-root-env-var
	# Saves the `cloudogu/k8s-ces-setup:version` image into a file into the K8s root path to be available on all nodes.
	docker save ${IMAGE} -o ${K8S_CLUSTER_ROOT}/image.tar

.PHONY: image-import
image-import: ${K8S_CLUSTER_ROOT}/image.tar
    # Imports the currently available image `cloudogu/k8s-ces-setup:version` into the K8s cluster for all nodes.
	@echo "Import docker image of dogu into all K8s nodes..."
	@cd ${K8S_CLUSTER_ROOT} && \
		for node in $$(vagrant status --machine-readable | grep "state,running" | awk -F',' '{print $$2}'); \
		do  \
			echo "...$${node}"; \
			vagrant ssh $${node} -- -t "sudo k3s ctr images import /vagrant/image.tar"; \
		done;
	@echo "Done."
	rm ${K8S_CLUSTER_ROOT}/image.tar

.PHONY: check-k8s-cluster-root-env-var
check-k8s-cluster-root-env-var:
	@echo "Checking if env var K8S_CLUSTER_ROOT is set..."
	@bash -c export -p | grep K8S_CLUSTER_ROOT
	@echo "Done."

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
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMAGE}
	$(KUSTOMIZE) build config/default > ${K8S_RESOURCE_YAML}

.PHONY: k8s-generate
k8s-generate: ${K8S_RESOURCE_YAML} ## Create required k8s resources in ./dist/...
	@echo "Generating new kubernetes resources..."

.PHONY: k8s-apply
k8s-apply: k8s-generate ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cat ${K8S_RESOURCE_YAML} | kubectl apply -f -

.PHONY: k8s-delete
k8s-delete: k8s-generate ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	cat ${K8S_RESOURCE_YAML} | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

##@ Download Kubernetes Utility Tools

CONTROLLER_GEN = $(UTILITY_BIN_PATH)/controller-gen
.PHONY: controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.8.0)

KUSTOMIZE = $(UTILITY_BIN_PATH)/kustomize
.PHONY: kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4@v4.5.2)

ENVTEST = $(UTILITY_BIN_PATH)/setup-envtest
.PHONY: envtest
envtest: ## Download envtest-setup locally if necessary.
	$(call go-get-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)