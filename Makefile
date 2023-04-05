# Set these to the desired values
ARTIFACT_ID=k8s-dogu-operator
VERSION=0.28.0
## Image URL to use all building/pushing image targets
IMAGE_DEV=${K3CES_REGISTRY_URL_PREFIX}/${ARTIFACT_ID}:${VERSION}
IMAGE=cloudogu/${ARTIFACT_ID}:${VERSION}
GOTAG?=1.20.2
MAKEFILES_VERSION=7.5.0
LINT_VERSION?=v1.52.1
STAGE?=production

ADDITIONAL_CLEAN=dist-clean

include build/make/variables.mk
include build/make/self-update.mk
include build/make/dependencies-gomod.mk
include build/make/build.mk
include build/make/test-common.mk
include build/make/test-unit.mk
include build/make/static-analysis.mk
include build/make/clean.mk
include build/make/digital-signature.mk

K8S_RUN_PRE_TARGETS=install setup-etcd-port-forward
PRE_COMPILE=generate

K8S_RESOURCE_TEMP_FOLDER ?= $(TARGET_DIR)
K8S_PRE_GENERATE_TARGETS=k8s-create-temporary-resource template-stage template-dev-only-image-pull-policy template-log-level

include build/make/k8s-controller.mk

.PHONY: build-boot
build-boot: image-import k8s-apply kill-operator-pod ## Builds a new version of the dogu and deploys it into the K8s-EcoSystem.

##@ Controller specific targets

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	@echo "Generate manifests..."
	@$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	@make template-crd-labels
	@cp config/crd/bases/k8s.cloudogu.com_dogus.yaml api/v1/

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	@echo "Auto-generate deepcopy functions..."
	@$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

##@ Deployment

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	@$(KUSTOMIZE) build config/crd | kubectl delete --wait=false --ignore-not-found=true -f -
	@kubectl patch crd/dogus.k8s.cloudogu.com -p '{"metadata":{"finalizers":[]}}' --type=merge || true

.PHONY: setup-etcd-port-forward
setup-etcd-port-forward:
	kubectl -n ${NAMESPACE} port-forward etcd-0 4001:2379 &

.PHONY: template-crd-labels
template-crd-labels: kustomize
	@$(KUSTOMIZE) build config/labels -o config/crd/bases/k8s.cloudogu.com_dogus.yaml

.PHONY: template-stage
template-stage:
	@echo "Setting STAGE env in deployment to ${STAGE}!"
	@$(BINARY_YQ) -i e "(select(.kind == \"Deployment\").spec.template.spec.containers[]|select(.image == \"*$(ARTIFACT_ID)*\").env[]|select(.name==\"STAGE\").value)=\"${STAGE}\"" $(K8S_RESOURCE_TEMP_YAML)

.PHONY: template-log-level
template-log-level:
	@echo "Setting LOG_LEVEL env in deployment to ${LOG_LEVEL}!"
	@$(BINARY_YQ) -i e "(select(.kind == \"Deployment\").spec.template.spec.containers[]|select(.image == \"*$(ARTIFACT_ID)*\").env[]|select(.name==\"LOG_LEVEL\").value)=\"${LOG_LEVEL}\"" $(K8S_RESOURCE_TEMP_YAML)

.PHONY: template-dev-only-image-pull-policy
template-dev-only-image-pull-policy: $(BINARY_YQ)
	@echo "Setting pull policy to always!"
	@$(BINARY_YQ) -i e "(select(.kind == \"Deployment\").spec.template.spec.containers[]|select(.image == \"*$(ARTIFACT_ID)*\").imagePullPolicy)=\"Always\"" $(K8S_RESOURCE_TEMP_YAML)

.PHONY: kill-operator-pod
kill-operator-pod:
	@echo "Restarting k8s-dogu-operator!"
	@kubectl -n ${NAMESPACE} delete pods -l 'app.kubernetes.io/name=k8s-dogu-operator'

##@ Debug

.PHONY: print-debug-info
print-debug-info: ## Generates info and the list of environment variables required to start the operator in debug mode.
	@echo "The target generates a list of env variables required to start the operator in debug mode. These can be pasted directly into the 'go build' run configuration in IntelliJ to run and debug the operator on-demand."
	@echo "STAGE=$(STAGE);LOG_LEVEL=$(LOG_LEVEL);KUBECONFIG=$(KUBECONFIG);NAMESPACE=$(NAMESPACE);DOGU_REGISTRY_ENDPOINT=$(DOGU_REGISTRY_ENDPOINT);DOGU_REGISTRY_USERNAME=$(DOGU_REGISTRY_USERNAME);DOGU_REGISTRY_PASSWORD=$(DOGU_REGISTRY_PASSWORD);DOCKER_REGISTRY={\"auths\":{\"$(docker_registry_server)\":{\"username\":\"$(docker_registry_username)\",\"password\":\"$(docker_registry_password)\",\"email\":\"ignore@me.com\",\"auth\":\"ignoreMe\"}}}"

##@ Mockery

MOCKERY_BIN=${UTILITY_BIN_PATH}/mockery
MOCKERY_VERSION=v2.20.0

${MOCKERY_BIN}: ${UTILITY_BIN_PATH}
	$(call go-get-tool,$(MOCKERY_BIN),github.com/vektra/mockery/v2@$(MOCKERY_VERSION))

mocks: ${MOCKERY_BIN} ## Generate all mocks for the dogu operator.
# Mockery respects .mockery.yaml in the project root
	@${MOCKERY_BIN} --output internal/cloudogu/mocks --srcpkg github.com/cloudogu/k8s-dogu-operator/internal/cloudogu --all
	@${MOCKERY_BIN} --output internal/thirdParty/mocks --srcpkg github.com/cloudogu/k8s-dogu-operator/internal/thirdParty --all
	@echo "Mocks successfully created."
