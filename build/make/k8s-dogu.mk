# This script required the k8s.mk script
include $(WORKDIR)/build/make/k8s.mk

## Variables

# Path to the dogu json of the dogu
DOGU_JSON_FILE=$(WORKDIR)/dogu.json
# Name of the dogu is extracted from the dogu.json
ARTIFACT_ID=$(shell yq -e ".Name" $(DOGU_JSON_FILE) | sed "s|.*/||g")
# Namespace of the dogu is extracted from the dogu.json
ARTIFACT_NAMESPACE=$(shell yq -e ".Name" $(DOGU_JSON_FILE) | sed "s|/.*||g")
# Namespace of the dogu is extracted from the dogu.json
VERSION=$(shell yq -e ".Version" $(DOGU_JSON_FILE))
# Image of the dogu is extracted from the dogu.json
IMAGE=$(shell yq -e ".Image" $(DOGU_JSON_FILE)):$(VERSION)

##@ K8s - EcoSystem

.PHONY: build
build: k8s-delete image-import install-dogu-descriptor k8s-apply ## Builds a new version of the dogu and deploys it into the K8s-EcoSystem.

##@ K8s - Dogu - Resource

# The additional k8s yaml files
K8S_RESOURCE_PRODUCTIVE_FOLDER ?= $(WORKDIR)/k8s
K8S_RESOURCE_PRODUCTIVE_YAML ?= $(K8S_RESOURCE_PRODUCTIVE_FOLDER)/$(ARTIFACT_ID).yaml
K8S_RESOURCE_DOGU_CR_TEMPLATE_YAML ?= $(WORKDIR)/build/make/k8s-dogu.tpl
# The pre generation script creates a k8s resource yaml containing the dogu crd and the content from the k8s folder.
.PHONY: k8s-create-temporary-resource
 k8s-create-temporary-resource:
	@echo "Generating temporary k8s resources $(K8S_RESOURCE_TEMP_YAML)..."
	@rm -f $(K8S_RESOURCE_TEMP_YAML)
	@test -f $(K8S_RESOURCE_PRODUCTIVE_YAML) && (cp $(K8S_RESOURCE_PRODUCTIVE_YAML) $(K8S_RESOURCE_TEMP_YAML)) || (touch $(K8S_RESOURCE_TEMP_YAML))
	@echo "---" >> $(K8S_RESOURCE_TEMP_YAML)
	@sed "s|NAMESPACE|$(ARTIFACT_NAMESPACE)|g" $(K8S_RESOURCE_DOGU_CR_TEMPLATE_YAML) | sed "s|NAME|$(ARTIFACT_ID)|g"  | sed "s|VERSION|$(VERSION)|g" >> $(K8S_RESOURCE_TEMP_YAML)
	@echo "Done."

##@ K8s - Dogu

.PHONY: install-dogu-descriptor
install-dogu-descriptor: ## Installs a configmap with current dogu.json into the cluster.
	@echo "Generate configmap from dogu.json..."
	@kubectl create configmap "$(ARTIFACT_ID)-descriptor" --from-file=$(DOGU_JSON_FILE) --dry-run=client -o yaml | kubectl apply -f -
	@echo "Done."