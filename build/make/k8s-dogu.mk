# Variables
# Path to the dogu json of the dogu
DOGU_JSON_FILE=${WORKDIR}/dogu.json
DOGU_JSON_DEV_FILE=${WORKDIR}/${TARGET_DIR}/dogu.json
# Name of the dogu is extracted from the dogu.json
ARTIFACT_ID=$(shell $(BINARY_YQ) -oy -e ".Name" $(DOGU_JSON_FILE) | sed "s|.*/||g")
# Namespace of the dogu is extracted from the dogu.json
ARTIFACT_NAMESPACE=$(shell $(BINARY_YQ) -oy -e ".Name" $(DOGU_JSON_FILE) | sed "s|/.*||g")
# Version of the dogu is extracted from the dogu.json
VERSION=$(shell $(BINARY_YQ) -oy -e ".Version" $(DOGU_JSON_FILE))
# Image of the dogu is extracted from the dogu.json
IMAGE=$(shell $(BINARY_YQ) -oy -e ".Image" $(DOGU_JSON_FILE)):$(VERSION)
IMAGE_DEV_WITHOUT_TAG=$(shell $(BINARY_YQ) -oy -e ".Image" $(DOGU_JSON_FILE) | sed "s|registry\.cloudogu\.com\(.\+\)|${K3CES_REGISTRY_URL_PREFIX}\1|g")
IMAGE_DEV=${IMAGE_DEV_WITHOUT_TAG}

include $(BUILD_DIR)/make/k8s.mk

##@ K8s - EcoSystem

.PHONY: build
build: image-import install-dogu-descriptor create-dogu-resource apply-dogu-resource ## Builds a new version of the dogu and deploys it into the K8s-EcoSystem.

##@ K8s - Dogu - Resource

# The additional k8s yaml files
K8S_RESOURCE_PRODUCTIVE_FOLDER ?= $(WORKDIR)/k8s
K8S_RESOURCE_PRODUCTIVE_YAML ?= $(K8S_RESOURCE_PRODUCTIVE_FOLDER)/$(ARTIFACT_ID).yaml
K8S_RESOURCE_DOGU_CR_TEMPLATE_YAML ?= $(BUILD_DIR)/make/k8s-dogu.tpl
K8S_RESOURCE_DOGU ?= $(K8S_RESOURCE_TEMP_FOLDER)/$(ARTIFACT_ID).yaml
# The pre generation script creates a k8s resource yaml containing the dogu crd and the content from the k8s folder.
.PHONY: create-dogu-resource
create-dogu-resource: ${BINARY_YQ} $(K8S_RESOURCE_TEMP_FOLDER)
	@echo "Generating temporary K8s resources $(K8S_RESOURCE_DOGU)..."
	@rm -f $(K8S_RESOURCE_DOGU)
	@sed "s|NAMESPACE|$(ARTIFACT_NAMESPACE)|g" $(K8S_RESOURCE_DOGU_CR_TEMPLATE_YAML) | sed "s|NAME|$(ARTIFACT_ID)|g"  | sed "s|VERSION|$(VERSION)|g" >> $(K8S_RESOURCE_DOGU)
	@echo "Done."

.PHONY: apply-dogu-resource
apply-dogu-resource:
	@kubectl apply -f "$(K8S_RESOURCE_DOGU)"

##@ K8s - Dogu

.PHONY: install-dogu-descriptor
install-dogu-descriptor: ${BINARY_YQ} $(TARGET_DIR) ## Installs a configmap with current dogu.json into the cluster.
	@echo "Generate configmap from dogu.json..."
	@$(BINARY_YQ) -oj ".Image=\"${IMAGE_DEV_WITHOUT_TAG}\"" ${DOGU_JSON_FILE} > ${DOGU_JSON_DEV_FILE}
	@kubectl create configmap "$(ARTIFACT_ID)-descriptor" --from-file=$(DOGU_JSON_DEV_FILE) --dry-run=client -o yaml | kubectl apply -f - --namespace=${NAMESPACE}
	@echo "Done."
