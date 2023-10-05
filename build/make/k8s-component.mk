DEV_VERSION?=${VERSION}-dev
## Image URL to use all building/pushing image targets
IMAGE_DEV?=${K3CES_REGISTRY_URL_PREFIX}/${ARTIFACT_ID}:${DEV_VERSION}

include $(WORKDIR)/build/make/k8s.mk

BINARY_HELM = $(UTILITY_BIN_PATH)/helm
BINARY_HELM_VERSION?=v3.12.0-dev.1.0.20230817154107-a749b663101d
BINARY_HELM_ADDITIONAL_PUSH_ARGS?=--plain-http
BINARY_HELM_ADDITIONAL_PACK_ARGS?=
BINARY_HELM_ADDITIONAL_UNINST_ARGS?=
BINARY_HELM_ADDITIONAL_UPGR_ARGS?=

K8S_HELM_TARGET ?= $(K8S_RESOURCE_TEMP_FOLDER)/helm
K8S_HELM_RESSOURCES ?= k8s/helm
K8S_HELM_RELEASE_TGZ=${K8S_HELM_TARGET}/${ARTIFACT_ID}-${VERSION}.tgz
K8S_HELM_DEV_RELEASE_TGZ=${K8S_HELM_TARGET}/${ARTIFACT_ID}-${DEV_VERSION}.tgz
K8S_HELM_ARTIFACT_NAMESPACE?=k8s

K8S_RESOURCE_COMPONENT ?= "${K8S_RESOURCE_TEMP_FOLDER}/component-${ARTIFACT_ID}-${VERSION}.yaml"
K8S_RESOURCE_COMPONENT_CR_TEMPLATE_YAML ?= $(WORKDIR)/build/make/k8s-component.tpl

##@ K8s - Helm general
.PHONY: helm-init-chart
helm-init-chart: ${BINARY_HELM} ## Creates a Chart.yaml-template with zero values
	@echo "Initialize ${K8S_HELM_RESSOURCES}/Chart.yaml..."
	@mkdir -p ${K8S_HELM_RESSOURCES}/tmp/
	@${BINARY_HELM} create ${K8S_HELM_RESSOURCES}/tmp/${ARTIFACT_ID}
	@cp ${K8S_HELM_RESSOURCES}/tmp/${ARTIFACT_ID}/Chart.yaml ${K8S_HELM_RESSOURCES}/
	@rm -dr ${K8S_HELM_RESSOURCES}/tmp
	@sed -i 's/appVersion: ".*"/appVersion: "0.0.0-replaceme"/' ${K8S_HELM_RESSOURCES}/Chart.yaml
	@sed -i 's/version: .*/version: 0.0.0-replaceme/' ${K8S_HELM_RESSOURCES}/Chart.yaml

.PHONY: helm-generate-chart
helm-generate-chart: k8s-generate ${K8S_HELM_TARGET}/Chart.yaml ## Generates the final helm chart.

.PHONY: ${K8S_HELM_TARGET}/Chart.yaml
${K8S_HELM_TARGET}/Chart.yaml: $(K8S_RESOURCE_TEMP_FOLDER) k8s-generate
	@echo "Generate helm chart..."
	@rm -drf ${K8S_HELM_TARGET}  # delete folder, so the chart is newly created.
	@mkdir -p ${K8S_HELM_TARGET}/templates
	@cp $(K8S_RESOURCE_TEMP_YAML) ${K8S_HELM_TARGET}/templates
	@${BINARY_YQ} 'select(document_index != (select(.kind == "CustomResourceDefinition") | document_index))' $(K8S_RESOURCE_TEMP_YAML) > ${K8S_HELM_TARGET}/templates/$(ARTIFACT_ID)_$(VERSION).yaml # select all documents without the CRD
	@sed -i "s/'{{ .Namespace }}'/'{{ .Release.Namespace }}'/" ${K8S_HELM_TARGET}/templates/$(ARTIFACT_ID)_$(VERSION).yaml
	@cp -r ${K8S_HELM_RESSOURCES}/** ${K8S_HELM_TARGET}
	@if [[ ${STAGE} == "development" ]]; then \
  	  sed -i 's/appVersion: "0.0.0-replaceme"/appVersion: '$(DEV_VERSION)'/' ${K8S_HELM_TARGET}/Chart.yaml; \
  	  sed -i 's/version: 0.0.0-replaceme/version:  '$(DEV_VERSION)'/' ${K8S_HELM_TARGET}/Chart.yaml; \
  	else \
  	  sed -i 's/appVersion: "0.0.0-replaceme"/appVersion: "${VERSION}"/' ${K8S_HELM_TARGET}/Chart.yaml; \
      sed -i 's/version: 0.0.0-replaceme/version: ${VERSION}/' ${K8S_HELM_TARGET}/Chart.yaml; \
    fi

##@ K8s - Helm dev targets

.PHONY: helm-generate
helm-generate: helm-generate-chart ## Generates the final helm chart with dev-urls.

.PHONY: helm-apply
helm-apply: ${BINARY_HELM} check-k8s-namespace-env-var image-import helm-generate $(K8S_POST_GENERATE_TARGETS) ## Generates and installs the helm chart.
	@echo "Apply generated helm chart"
	@${BINARY_HELM} upgrade -i ${ARTIFACT_ID} ${K8S_HELM_TARGET} ${BINARY_HELM_ADDITIONAL_UPGR_ARGS} --namespace ${NAMESPACE}

.PHONY: helm-delete
helm-delete: ${BINARY_HELM} check-k8s-namespace-env-var ## Uninstalls the current helm chart.
	@echo "Uninstall helm chart"
	@${BINARY_HELM} uninstall ${ARTIFACT_ID} --namespace=${NAMESPACE} ${BINARY_HELM_ADDITIONAL_UNINST_ARGS} || true

.PHONY: helm-reinstall
helm-reinstall: helm-delete helm-apply ## Uninstalls the current helm chart and reinstalls it.

.PHONY: helm-chart-import
helm-chart-import: check-all-vars check-k8s-artifact-id helm-generate-chart helm-package-release image-import ## Imports the currently available chart into the cluster-local registry.
	@if [[ ${STAGE} == "development" ]]; then \
		echo "Import ${K8S_HELM_DEV_RELEASE_TGZ} into K8s cluster ${K3CES_REGISTRY_URL_PREFIX}..."; \
		${BINARY_HELM} push ${K8S_HELM_DEV_RELEASE_TGZ} oci://${K3CES_REGISTRY_URL_PREFIX}/${K8S_HELM_ARTIFACT_NAMESPACE} ${BINARY_HELM_ADDITIONAL_PUSH_ARGS}; \
	else \
	  	echo "Import ${K8S_HELM_RELEASE_TGZ} into K8s cluster ${K3CES_REGISTRY_URL_PREFIX}..."; \
        ${BINARY_HELM} push ${K8S_HELM_RELEASE_TGZ} oci://${K3CES_REGISTRY_URL_PREFIX}/${K8S_HELM_ARTIFACT_NAMESPACE} ${BINARY_HELM_ADDITIONAL_PUSH_ARGS}; \
    fi
	@echo "Done."

##@ K8s - Helm release targets

.PHONY: helm-generate-release
helm-generate-release: ${K8S_HELM_TARGET}/templates/$(ARTIFACT_ID)_$(VERSION).yaml ## Generates the final helm chart with release urls.

${K8S_HELM_TARGET}/templates/$(ARTIFACT_ID)_$(VERSION).yaml: $(K8S_PRE_GENERATE_TARGETS) ${K8S_HELM_TARGET}/Chart.yaml
	@sed -i "s/'{{ .Namespace }}'/'{{ .Release.Namespace }}'/" ${K8S_HELM_TARGET}/templates/$(ARTIFACT_ID)_$(VERSION).yaml

.PHONY: helm-package-release
helm-package-release: ${BINARY_HELM} helm-delete-existing-tgz ${K8S_HELM_RELEASE_TGZ} ## Generates and packages the helm chart with release urls.

.PHONY: helm-delete-existing-tgz
helm-delete-existing-tgz: ## Remove an existing Helm package.
# remove
	@rm -f ${K8S_HELM_RELEASE_TGZ}*

${K8S_HELM_RELEASE_TGZ}: ${BINARY_HELM} ${K8S_HELM_TARGET}/templates/$(ARTIFACT_ID)_$(VERSION).yaml helm-generate-chart $(K8S_POST_GENERATE_TARGETS) ## Generates and packages the helm chart with release urls.
	@echo "Package generated helm chart"
	@${BINARY_HELM} package ${K8S_HELM_TARGET} -d ${K8S_HELM_TARGET} ${BINARY_HELM_ADDITIONAL_PACK_ARGS}

${BINARY_HELM}: $(UTILITY_BIN_PATH) ## Download helm locally if necessary.
	$(call go-get-tool,$(BINARY_HELM),helm.sh/helm/v3/cmd/helm@${BINARY_HELM_VERSION})

##@ K8s - Component dev targets

.PHONY: component-generate
component-generate: ${K8S_RESOURCE_TEMP_FOLDER} ## Generate the component yaml resource.
	@echo "Generating temporary K8s component resource: ${K8S_RESOURCE_COMPONENT}"
	@if [[ ${STAGE} == "development" ]]; then \
		sed "s|NAMESPACE|$(K8S_HELM_ARTIFACT_NAMESPACE)|g" "${K8S_RESOURCE_COMPONENT_CR_TEMPLATE_YAML}" | sed "s|NAME|$(ARTIFACT_ID)|g"  | sed "s|VERSION|$(DEV_VERSION)|g" > "${K8S_RESOURCE_COMPONENT}"; \
	else \
		sed "s|NAMESPACE|$(K8S_HELM_ARTIFACT_NAMESPACE)|g" "${K8S_RESOURCE_COMPONENT_CR_TEMPLATE_YAML}" | sed "s|NAME|$(ARTIFACT_ID)|g"  | sed "s|VERSION|$(VERSION)|g" > "${K8S_RESOURCE_COMPONENT}"; \
	fi

.PHONY: component-apply
component-apply: check-k8s-namespace-env-var image-import helm-generate helm-chart-import component-generate $(K8S_POST_GENERATE_TARGETS) ## Applies the component yaml resource to the actual defined context.
	@kubectl apply -f "${K8S_RESOURCE_COMPONENT}" --namespace="${NAMESPACE}"
	@echo "Done."

.PHONY: component-delete
component-delete: check-k8s-namespace-env-var component-generate $(K8S_POST_GENERATE_TARGETS) ## Deletes the component yaml resource from the actual defined context.
	@kubectl delete -f "${K8S_RESOURCE_COMPONENT}" --namespace="${NAMESPACE}" || true
	@echo "Done."

.PHONY: component-reinstall
component-reinstall: component-delete  component-apply ## Reinstalls the component yaml resource from the actual defined context.