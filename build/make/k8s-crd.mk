ARTIFACT_CRD_ID=$(ARTIFACT_ID)-crd
DEV_CRD_VERSION?=${VERSION}-dev
K8S_HELM_CRD_TARGET ?= $(K8S_RESOURCE_TEMP_FOLDER)/helm-crd
K8S_HELM_CRD_RESSOURCES ?= k8s/helm-crd
K8S_HELM_CRD_RELEASE_TGZ=${K8S_HELM_CRD_TARGET}/${ARTIFACT_CRD_ID}-${VERSION}.tgz
K8S_HELM_CRD_DEV_RELEASE_TGZ=${K8S_HELM_CRD_TARGET}/${ARTIFACT_CRD_ID}-${DEV_CRD_VERSION}.tgz

K8S_RESOURCE_CRD_COMPONENT ?= "${K8S_RESOURCE_TEMP_FOLDER}/component-${ARTIFACT_CRD_ID}-${VERSION}.yaml"
K8S_RESOURCE_COMPONENT_CR_TEMPLATE_YAML ?= $(WORKDIR)/build/make/k8s-component.tpl

##@ K8s - CRD targets

.PHONY: crd-helm-generate-chart ## Generates the helm crd-chart
crd-helm-generate-chart: ${BINARY_YQ} $(K8S_RESOURCE_TEMP_FOLDER) k8s-generate
	@echo "Generate helm crd-chart..."
	@rm -drf ${K8S_HELM_CRD_TARGET}  # delete folder, so the chart is newly created.
	@mkdir -p ${K8S_HELM_CRD_TARGET}/templates
	@cp -r ${K8S_HELM_CRD_RESSOURCES}/** ${K8S_HELM_CRD_TARGET}
	@${BINARY_YQ} 'select(.kind == "CustomResourceDefinition")' $(K8S_RESOURCE_TEMP_YAML) > ${K8S_HELM_CRD_TARGET}/templates/$(ARTIFACT_CRD_ID)_$(VERSION).yaml
	@sed -i 's/name: artifact-crd-replaceme/name: ${ARTIFACT_CRD_ID}/' ${K8S_HELM_CRD_TARGET}/Chart.yaml
	@if [[ ${STAGE} == "development" ]]; then \
	  sed -i 's/appVersion: "0.0.0-replaceme"/appVersion: "${DEV_CRD_VERSION}"/' ${K8S_HELM_CRD_TARGET}/Chart.yaml; \
      sed -i 's/version: 0.0.0-replaceme/version: ${DEV_CRD_VERSION}/' ${K8S_HELM_CRD_TARGET}/Chart.yaml; \
	else \
	  sed -i 's/appVersion: "0.0.0-replaceme"/appVersion: "${VERSION}"/' ${K8S_HELM_CRD_TARGET}/Chart.yaml; \
      sed -i 's/version: 0.0.0-replaceme/version: ${VERSION}/' ${K8S_HELM_CRD_TARGET}/Chart.yaml; \
	fi

.PHONY: crd-helm-apply
crd-helm-apply: ${BINARY_HELM} check-k8s-namespace-env-var crd-helm-generate-chart $(K8S_POST_GENERATE_TARGETS) ## Generates and installs the helm crd-chart.
	@echo "Apply generated helm crd-chart"
	@${BINARY_HELM} upgrade -i ${ARTIFACT_CRD_ID} ${K8S_HELM_CRD_TARGET} ${BINARY_HELM_ADDITIONAL_UPGR_ARGS} --namespace ${NAMESPACE}

.PHONY: crd-helm-delete
crd-helm-delete: ${BINARY_HELM} check-k8s-namespace-env-var ## Uninstalls the current helm crd-chart.
	@echo "Uninstall helm crd-chart"
	@${BINARY_HELM} uninstall ${ARTIFACT_CRD_ID} --namespace=${NAMESPACE} ${BINARY_HELM_ADDITIONAL_UNINST_ARGS} || true

.PHONY: crd-helm-package
crd-helm-package: ${BINARY_HELM} crd-helm-delete-existing-tgz ${K8S_HELM_CRD_RELEASE_TGZ} ## Generates and packages the helm crd-chart.

.PHONY: crd-helm-delete-existing-tgz
crd-helm-delete-existing-tgz: ## Remove an existing Helm crd-package.
	@rm -f ${K8S_HELM_CRD_RELEASE_TGZ}*

${K8S_HELM_CRD_RELEASE_TGZ}: ${BINARY_HELM} crd-helm-generate-chart $(K8S_POST_GENERATE_TARGETS) ## Generates and packages the helm crd-chart.
	@echo "Package generated helm crd-chart"
	@${BINARY_HELM} package ${K8S_HELM_CRD_TARGET} -d ${K8S_HELM_CRD_TARGET} ${BINARY_HELM_ADDITIONAL_PACK_ARGS}

.PHONY: crd-helm-chart-import
crd-helm-chart-import: check-all-vars check-k8s-artifact-id crd-helm-generate-chart crd-helm-package ## Imports the currently available crd-chart into the cluster-local registry.
	@if [[ ${STAGE} == "development" ]]; then \
		echo "Import ${K8S_HELM_CRD_DEV_RELEASE_TGZ} into K8s cluster ${K3CES_REGISTRY_URL_PREFIX}..."; \
		${BINARY_HELM} push ${K8S_HELM_CRD_DEV_RELEASE_TGZ} oci://${K3CES_REGISTRY_URL_PREFIX}/${K8S_HELM_ARTIFACT_NAMESPACE} ${BINARY_HELM_ADDITIONAL_PUSH_ARGS}; \
	else \
	  	echo "Import ${K8S_HELM_CRD_RELEASE_TGZ} into K8s cluster ${K3CES_REGISTRY_URL_PREFIX}..."; \
        ${BINARY_HELM} push ${K8S_HELM_CRD_RELEASE_TGZ} oci://${K3CES_REGISTRY_URL_PREFIX}/${K8S_HELM_ARTIFACT_NAMESPACE} ${BINARY_HELM_ADDITIONAL_PUSH_ARGS}; \
    fi
	@echo "Done."

.PHONY: crd-component-generate
crd-component-generate: ${K8S_RESOURCE_TEMP_FOLDER} ## Generate the crd-component yaml resource.
	@echo "Generating temporary K8s crd-component resource: ${K8S_RESOURCE_CRD_COMPONENT}"
	@if [[ ${STAGE} == "development" ]]; then \
		sed "s|NAMESPACE|$(K8S_HELM_ARTIFACT_NAMESPACE)|g" "${K8S_RESOURCE_COMPONENT_CR_TEMPLATE_YAML}" | sed "s|NAME|$(ARTIFACT_CRD_ID)|g"  | sed "s|VERSION|$(DEV_CRD_VERSION)|g" > "${K8S_RESOURCE_CRD_COMPONENT}"; \
	else \
		sed "s|NAMESPACE|$(K8S_HELM_ARTIFACT_NAMESPACE)|g" "${K8S_RESOURCE_COMPONENT_CR_TEMPLATE_YAML}" | sed "s|NAME|$(ARTIFACT_CRD_ID)|g"  | sed "s|VERSION|$(VERSION)|g" > "${K8S_RESOURCE_CRD_COMPONENT}"; \
	fi

.PHONY: crd-component-apply
crd-component-apply: check-k8s-namespace-env-var crd-helm-chart-import crd-component-generate $(K8S_POST_GENERATE_TARGETS) ## Applies the crd-component yaml resource to the actual defined context.
	@kubectl apply -f "${K8S_RESOURCE_CRD_COMPONENT}" --namespace="${NAMESPACE}"
	@echo "Done."

.PHONY: crd-component-delete
crd-component-delete: check-k8s-namespace-env-var crd-component-generate $(K8S_POST_GENERATE_TARGETS) ## Deletes the crd-component yaml resource from the actual defined context.
	@kubectl delete -f "${K8S_RESOURCE_CRD_COMPONENT}" --namespace="${NAMESPACE}" || true
	@echo "Done."
