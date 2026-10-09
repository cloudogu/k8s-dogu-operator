# Set these to the desired values
ARTIFACT_ID=k8s-dogu-operator
VERSION=3.31.1

IMAGE=cloudogu/${ARTIFACT_ID}:${VERSION}
GOTAG=1.27.2
LINT_VERSION=v2.13.2
MAKEFILES_VERSION=11.0.0

PRE_COMPILE = generate-deepcopy
K8S_COMPONENT_SOURCE_VALUES = ${HELM_SOURCE_DIR}/values.yaml
K8S_COMPONENT_TARGET_VALUES = ${HELM_TARGET_DIR}/values.yaml
HELM_PRE_GENERATE_TARGETS = helm-values-update-image-version
HELM_POST_GENERATE_TARGETS = helm-values-replace-image-repo template-stage template-log-level template-image-pull-policy
IMAGE_IMPORT_TARGET=image-import
CHECK_VAR_TARGETS=check-all-vars

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
include build/make/vulnerability-scan.mk

HELMFILE_IMAGE_VERSION=0.1.0

.PHONY: build-helmfile-image
build-helmfile-image:
	@docker build -f helmfile.Dockerfile . -t "registry.cloudogu.com/internal/helmfile:${HELMFILE_IMAGE_VERSION}"

.PHONY: push-helmfile-image
push-helmfile-image: build-helmfile-image
	@docker push "registry.cloudogu.com/internal/helmfile:${HELMFILE_IMAGE_VERSION}"

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

##@ Mock DCC (V3)

MOCK_DCC_DIR := dev/mock-dcc-v3
MOCK_DCC_ENDPOINT := http://test-dccv3:8080/api/v3/dogus

.PHONY: mock-dcc-v3
mock-dcc-v3: ## Deploys a mock Dogu V3 DCC (nginx) for 'testing/nexus' and points the dogu-registry-v3 secret at it.
	@echo "Backing up current dogu-registry-v3 endpoint..."
	@if [ ! -f $(MOCK_DCC_DIR)/.endpoint.bak ]; then \
		if kubectl -n ${NAMESPACE} get secret dogu-registry-v3 >/dev/null 2>&1; then \
			kubectl -n ${NAMESPACE} get secret dogu-registry-v3 -o jsonpath='{.data.endpoint}' | base64 -d > $(MOCK_DCC_DIR)/.endpoint.bak; \
			echo "  saved existing endpoint to $(MOCK_DCC_DIR)/.endpoint.bak"; \
		else \
			printf '__ABSENT__' > $(MOCK_DCC_DIR)/.endpoint.bak; \
			echo "  secret did not exist; recorded __ABSENT__"; \
		fi; \
	else \
		echo "  backup already exists, keeping it"; \
	fi
	@echo "Creating/updating configmap test-v3-dogus..."
	@kubectl -n ${NAMESPACE} create configmap test-v3-dogus \
		--from-file=3.86.2-8=$(MOCK_DCC_DIR)/nexus.json \
		--from-file=3.86.2-9=$(MOCK_DCC_DIR)/nexus-not-found.json \
		--dry-run=client -o yaml | kubectl -n ${NAMESPACE} apply -f -
	@echo "Deploying mock DCC (pod/service/networkpolicy)..."
	@echo "  recreating pod so it picks up the current descriptors (mounted configmaps resync lazily)"
	@kubectl -n ${NAMESPACE} delete pod test-dccv3 --ignore-not-found
	@kubectl -n ${NAMESPACE} apply -f $(MOCK_DCC_DIR)/dcc.yaml
	@echo "Ensuring dogu-registry-v3 secret exists..."
	@kubectl -n ${NAMESPACE} get secret dogu-registry-v3 >/dev/null 2>&1 || \
		kubectl -n ${NAMESPACE} create secret generic dogu-registry-v3 \
			--from-literal=endpoint=$(MOCK_DCC_ENDPOINT) \
			--from-literal=username=mock \
			--from-literal=password=mock \
			--from-literal=urlschema=default \
			--from-literal=insecureSkipVerify=true
	@echo "Patching dogu-registry-v3 endpoint to the mock..."
	@kubectl -n ${NAMESPACE} patch secret dogu-registry-v3 -p '{"stringData":{"endpoint": "$(MOCK_DCC_ENDPOINT)"}}'
	@echo ""
	@echo "Mock DCC ready. Apply a scenario (one at a time):"
	@echo "  404 dogu-registry: kubectl -n ${NAMESPACE} apply -f $(MOCK_DCC_DIR)/dogu-404-dogu-registry.yaml"
	@echo "  404 helm-registry: kubectl -n ${NAMESPACE} apply -f $(MOCK_DCC_DIR)/dogu-404-helm-registry.yaml"
	@echo "  success:           kubectl -n ${NAMESPACE} apply -f $(MOCK_DCC_DIR)/dogu-success.yaml"

.PHONY: mock-dcc-v3-clean
mock-dcc-v3-clean: ## Removes the mock Dogu V3 DCC and restores the dogu-registry-v3 endpoint.
	@echo "Deleting mock DCC resources..."
	@kubectl -n ${NAMESPACE} delete -f $(MOCK_DCC_DIR)/dcc.yaml --ignore-not-found
	@kubectl -n ${NAMESPACE} delete configmap test-v3-dogus --ignore-not-found
	@if [ -f $(MOCK_DCC_DIR)/.endpoint.bak ]; then \
		ENDPOINT=$$(cat $(MOCK_DCC_DIR)/.endpoint.bak); \
		if [ "$$ENDPOINT" = "__ABSENT__" ]; then \
			echo "Secret was absent before mocking; deleting dogu-registry-v3..."; \
			kubectl -n ${NAMESPACE} delete secret dogu-registry-v3 --ignore-not-found; \
		else \
			echo "Restoring dogu-registry-v3 endpoint to $$ENDPOINT..."; \
			kubectl -n ${NAMESPACE} patch secret dogu-registry-v3 -p "{\"stringData\":{\"endpoint\": \"$$ENDPOINT\"}}"; \
		fi; \
		rm -f $(MOCK_DCC_DIR)/.endpoint.bak; \
	else \
		echo "WARNING: no endpoint backup found ($(MOCK_DCC_DIR)/.endpoint.bak); leaving secret untouched."; \
	fi
