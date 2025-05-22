# Set these to the desired values
ARTIFACT_ID=k8s-dogu-operator
VERSION=3.6.0

IMAGE=cloudogu/${ARTIFACT_ID}:${VERSION}
GOTAG=1.24.3
LINT_VERSION=v1.64.7
MAKEFILES_VERSION=9.9.1

PRE_COMPILE = generate-deepcopy
K8S_COMPONENT_SOURCE_VALUES = ${HELM_SOURCE_DIR}/values.yaml
K8S_COMPONENT_TARGET_VALUES = ${HELM_TARGET_DIR}/values.yaml
HELM_PRE_GENERATE_TARGETS = helm-values-update-image-version
HELM_POST_GENERATE_TARGETS = helm-values-replace-image-repo template-stage template-log-level template-image-pull-policy
IMAGE_IMPORT_TARGET=image-import
CHECK_VAR_TARGETS=check-all-vars

MOCKERY_VERSION=v2.53.3

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
include build/make/mocks.mk

.PHONY: mocks
mocks: ${MOCKERY_BIN} ${MOCKERY_YAML} ## target is used to generate mocks for all interfaces in a project.
	${MOCKERY_BIN}
	@echo "Mocks successfully created."

.PHONY: build-boot
build-boot: helm-apply kill-operator-pod ## Builds a new version of the dogu and deploys it into the K8s-EcoSystem.

.PHONY: helm-values-update-image-version
helm-values-update-image-version: $(BINARY_YQ)
	@echo "Updating the image version in source value.yaml to ${VERSION}..."
	@$(BINARY_YQ) -i e ".controllerManager.image.tag = \"${VERSION}\"" ${K8S_COMPONENT_SOURCE_VALUES}

.PHONY: helm-values-replace-image-repo
helm-values-replace-image-repo: $(BINARY_YQ)
	@if [[ ${STAGE} == "development" ]]; then \
      		echo "Setting dev image repo in target value.yaml!" ;\
    		$(BINARY_YQ) -i e ".controllerManager.image.registry=\"$(shell echo '${IMAGE_DEV}' | sed 's/\([^\/]*\)\/\(.*\)/\1/')\"" ${K8S_COMPONENT_TARGET_VALUES} ;\
    		$(BINARY_YQ) -i e ".controllerManager.image.repository=\"$(shell echo '${IMAGE_DEV}' | sed 's/\([^\/]*\)\/\(.*\)/\2/')\"" ${K8S_COMPONENT_TARGET_VALUES} ;\
    	fi

##@ Deployment

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
	@echo "STAGE=$(STAGE);LOG_LEVEL=$(LOG_LEVEL);KUBECONFIG=$(KUBECONFIG);NAMESPACE=$(NAMESPACE);DOGU_REGISTRY_ENDPOINT=$(DOGU_REGISTRY_ENDPOINT);DOGU_REGISTRY_USERNAME=$(DOGU_REGISTRY_USERNAME);DOGU_REGISTRY_PASSWORD=$(DOGU_REGISTRY_PASSWORD)"
