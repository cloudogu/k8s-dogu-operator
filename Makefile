# Set these to the desired values
ARTIFACT_ID=k8s-dogu-operator
VERSION=0.38.0

IMAGE=cloudogu/${ARTIFACT_ID}:${VERSION}
GOTAG=1.21
MAKEFILES_VERSION=9.0.0
LINT_VERSION=v1.52.1

K8S_RUN_PRE_TARGETS = install setup-etcd-port-forward
PRE_COMPILE = generate-deepcopy

K8S_COMPONENT_SOURCE_VALUES = ${HELM_SOURCE_DIR}/values.yaml
K8S_COMPONENT_TARGET_VALUES = ${HELM_TARGET_DIR}/values.yaml
CRD_SOURCE = ${HELM_CRD_SOURCE_DIR}/templates/k8s.cloudogu.com_dogus.yaml
HELM_PRE_APPLY_TARGETS = template-stage template-image-pull-policy template-log-level
CRD_POST_MANIFEST_TARGETS = crd-add-labels crd-copy-for-go-embedding
HELM_PRE_GENERATE_TARGETS = helm-values-update-image-version
HELM_POST_GENERATE_TARGETS = helm-values-replace-image-repo

include build/make/variables.mk
include build/make/self-update.mk
include build/make/dependencies-gomod.mk
include build/make/build.mk
include build/make/test-common.mk
include build/make/test-unit.mk
include build/make/static-analysis.mk
include build/make/clean.mk
include build/make/digital-signature.mk
include build/make/k8s-controller.mk

.PHONY: build-boot
build-boot: crd-helm-apply helm-apply kill-operator-pod ## Builds a new version of the dogu and deploys it into the K8s-EcoSystem.

.PHONY: crd-add-labels
crd-add-labels: $(BINARY_YQ)
	@echo "Adding labels to CRD..."
	@$(BINARY_YQ) -i e ".metadata.labels.app = \"ces\"" ${CRD_SOURCE}
	@$(BINARY_YQ) -i e ".metadata.labels.\"app.kubernetes.io/name\" = \"${ARTIFACT_ID}\"" ${CRD_SOURCE}

.PHONY: crd-copy-for-go-embedding
crd-copy-for-go-embedding:
	@echo "Copy CRD to api/v1/"
	@cp ${CRD_SOURCE} api/v1/

.PHONY: helm-values-update-image-version
helm-values-update-image-version: $(BINARY_YQ)
	@echo "Updating the image version in source value.yaml to ${VERSION}..."
	@$(BINARY_YQ) -i e ".controllerManager.image.tag = \"${VERSION}\"" ${K8S_COMPONENT_SOURCE_VALUES}

.PHONY: helm-values-replace-image-repo
helm-values-replace-image-repo: $(BINARY_YQ)
	@if [[ ${STAGE} == "development" ]]; then \
      		echo "Setting dev image repo in target value.yaml!" ;\
    		$(BINARY_YQ) -i e ".controllerManager.image.repository=\"${IMAGE_DEV}\"" ${K8S_COMPONENT_TARGET_VALUES} ;\
    	fi

##@ Deployment

.PHONY: setup-etcd-port-forward
setup-etcd-port-forward:
	kubectl -n ${NAMESPACE} port-forward etcd-0 4001:2379 &

.PHONY: template-stage
template-stage: $(BINARY_YQ)
	@if [[ ${STAGE} == "development" ]]; then \
  		echo "Setting STAGE env in deployment to ${STAGE}!" ;\
		$(BINARY_YQ) -i e ".controllerManager.env.stage=\"${STAGE}\"" ${K8S_COMPONENT_TARGET_VALUES} ;\
	fi

.PHONY: template-log-level
template-log-level: $(BINARY_YQ)
	@echo "Setting LOG_LEVEL env in deployment to ${LOG_LEVEL}!"
	@$(BINARY_YQ) -i e ".controllerManager.env.logLevel=\"${LOG_LEVEL}\"" ${K8S_COMPONENT_TARGET_VALUES}

.PHONY: template-image-pull-policy
template-image-pull-policy: $(BINARY_YQ)
	@if [[ ${STAGE} == "development" ]]; then \
  		echo "Setting PULL POLICY to always!" ;\
		$(BINARY_YQ) -i e ".controllerManager.imagePullPolicy=\"Always\"" ${K8S_COMPONENT_TARGET_VALUES} ;\
	fi

.PHONY: kill-operator-pod
kill-operator-pod:
	@echo "Restarting k8s-dogu-operator!"
	@kubectl -n ${NAMESPACE} delete pods -l 'app.kubernetes.io/name=${ARTIFACT_ID}'

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
