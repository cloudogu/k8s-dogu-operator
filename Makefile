# Set these to the desired values
ARTIFACT_ID=k8s-dogu-operator
VERSION=0.24.0
## Image URL to use all building/pushing image targets
IMAGE_DEV=${K3CES_REGISTRY_URL_PREFIX}/${ARTIFACT_ID}:${VERSION}
IMAGE=cloudogu/${ARTIFACT_ID}:${VERSION}
GOTAG?=1.18
MAKEFILES_VERSION=7.2.0
LINT_VERSION=v1.45.2
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
print-debug-info: ## Generates indo and the list of environment variables required to start the operator in debug mode.
	@echo "The target generates a list of env variables required to start the operator in debug mode. These can be pasted directly into the 'go build' run configuration in IntelliJ to run and debug the operator on-demand."
	@echo "STAGE=$(STAGE);LOG_LEVEL=$(LOG_LEVEL);KUBECONFIG=$(KUBECONFIG);NAMESPACE=$(NAMESPACE);DOGU_REGISTRY_ENDPOINT=$(DOGU_REGISTRY_ENDPOINT);DOGU_REGISTRY_USERNAME=$(DOGU_REGISTRY_USERNAME);DOGU_REGISTRY_PASSWORD=$(DOGU_REGISTRY_PASSWORD);DOCKER_REGISTRY={\"auths\":{\"$(docker_registry_server)\":{\"username\":\"$(docker_registry_username)\",\"password\":\"$(docker_registry_password)\",\"email\":\"ignore@me.com\",\"auth\":\"ignoreMe\"}}}"

##@ Mockery

MOCKERY_BIN=${UTILITY_BIN_PATH}/mockery
MOCKERY_VERSION=v2.15.0

${MOCKERY_BIN}: ${UTILITY_BIN_PATH}
	$(call go-get-tool,$(MOCKERY_BIN),github.com/vektra/mockery/v2@$(MOCKERY_VERSION))

mocks: ${MOCKERY_BIN} ## This target is used to generate all mocks for the dogu operator.
	@cd $(WORKDIR)/internal && ${MOCKERY_BIN} --all
	@echo "Mocks successfully created."

##-----------------------------
K3D = $(UTILITY_BIN_PATH)/k3d
.PHONY: k3d
k3d: ## Download k3d locally if necessary.
	$(call go-get-tool,$(K3D),github.com/rancher/k3d@latest)

K3D_IMAGE="rancher/k3s:v1.21.2-k3s1"
K3D_CLUSTER_NAME="itest"
K3D_NAMESPACE="ecosystem"
SETUP_TAG="develop"

.PHONY: k3d-setup-ces-cluster
k3d-setup-ces-cluster: ${K3D} k3d-create-cluster k3d-create-namespace k3d-create-docker-registry-secret k3d-create-dogu-registry-secret k3d-apply-setup

.PHONY: k3d-create-cluster
k3d-create-cluster: ${K3D}
	@k3d cluster create "${K3D_CLUSTER_NAME}" --image="${K3D_IMAGE}"

.PHONY: k3d-create-namespace
k3d-create-namespace:
	@KUBECONFIG=$(k3d kubeconfig write "${K3D_CLUSTER_NAME}") kubectl create namespace "${K3D_NAMESPACE}" || true

.PHONY: k3d-create-docker-registry-secret
k3d-create-docker-registry-secret:
	@KUBECONFIG=$(k3d kubeconfig write "${K3D_CLUSTER_NAME}") kubectl --namespace "${K3D_NAMESPACE}" create secret docker-registry k8s-dogu-operator-docker-registry \
    --docker-server="${docker_registry_server}" \
    --docker-username="${docker_registry_username}" \
    --docker-email="myemail@test.com" \
    --docker-password="${docker_registry_password}"

.PHONY: k3d-create-dogu-registry-secret
k3d-create-dogu-registry-secret:
	@KUBECONFIG=$(k3d kubeconfig write "${K3D_CLUSTER_NAME}") kubectl --namespace "${K3D_NAMESPACE}" create secret generic k8s-dogu-operator-dogu-registry \
	--from-literal=endpoint="${DOGU_REGISTRY_ENDPOINT}" \
	--from-literal=username="${DOGU_REGISTRY_USERNAME}" \
	--from-literal=password="${DOGU_REGISTRY_PASSWORD}" \
	--from-literal=urlschema="default"

.PHONY: k3d-create-nodemaster
k3d-create-nodemaster:
	@$(k3d kubeconfig write "${K3D_CLUSTER_NAME}") kubectl create configmap node-master-file --namespace "${K3D_NAMESPACE}" --from-literal=node_master=etcd."${K3D_NAMESPACE}".svc.cluster.local

.PHONY: k3d-apply-etcd
k3d-apply-etcd:
	@$(k3d kubeconfig write "${K3D_CLUSTER_NAME}") kubectl apply --namespace "${K3D_NAMESPACE}" -f https://raw.githubusercontent.com/cloudogu/k8s-etcd/develop/manifests/etcd.yaml

SETUP_JSON_PATH=setup.json
SETUP_DIR=$(TMP_DIR)
TEMP_SETUP_PATCHED=$(SETUP_DIR)/setup_patched.yaml
TEMP_SETUP=$(SETUP_DIR)/setup.yaml
.PHONY: k3d-apply-setup
k3d-apply-setup: $(TMP_DIR)
	@$(k3d kubeconfig write "${K3D_CLUSTER_NAME}") kubectl apply -f https://raw.githubusercontent.com/cloudogu/k8s-ces-setup/${SETUP_TAG}/k8s/k8s-ces-setup-config.yaml --namespace  "${K3D_NAMESPACE}"
	@$(k3d kubeconfig write "${K3D_CLUSTER_NAME}") kubectl create configmap k8s-ces-setup-json --from-file="${SETUP_JSON_PATH}" --namespace "${K3D_NAMESPACE}"
	@wget -O "${TEMP_SETUP}" https://raw.githubusercontent.com/cloudogu/k8s-ces-setup/${SETUP_TAG}/k8s/k8s-ces-setup.yaml
	@yq "(select(.kind == \"ClusterRoleBinding\").subjects[]|select(.name == \"k8s-ces-setup\")).namespace=\"ecosystem\"" "${TEMP_SETUP}" | \
	yq "(select(.kind == \"ClusterRoleBinding\").subjects[]|select(.name == \"k8s-ces-setup-finisher\")).namespace=\"ecosystem\"" > "${TEMP_SETUP_PATCHED}"
	@$(k3d kubeconfig write "${K3D_CLUSTER_NAME}") kubectl apply -f "${TEMP_SETUP_PATCHED}" --namespace "${K3D_NAMESPACE}"
	@rm "${TEMP_SETUP}"
	@rm "${TEMP_SETUP_PATCHED}"

.PHONY: k3d-delete-cluster
k3d-delete-cluster: ${K3D}
	@k3d cluster delete "${K3D_CLUSTER_NAME}"