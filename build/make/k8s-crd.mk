ARTIFACT_CRD_ID=$(ARTIFACT_ID)-crd
DEV_CRD_VERSION?=${VERSION}-dev
K8S_HELM_CRD_TARGET?=$(K8S_RESOURCE_TEMP_FOLDER)/helm-crd
K8S_HELM_CRD_RESSOURCES?=k8s/helm-crd
K8S_HELM_CRD_RELEASE_TGZ=${K8S_HELM_CRD_TARGET}/${ARTIFACT_CRD_ID}-${VERSION}.tgz
K8S_HELM_CRD_DEV_RELEASE_TGZ=${K8S_HELM_CRD_TARGET}/${ARTIFACT_CRD_ID}-${DEV_CRD_VERSION}.tgz

K8S_RESOURCE_CRD_COMPONENT ?= "${K8S_RESOURCE_TEMP_FOLDER}/component-${ARTIFACT_CRD_ID}-${VERSION}.yaml"
K8S_RESOURCE_COMPONENT_CR_TEMPLATE_YAML ?= $(WORKDIR)/build/make/k8s-component.tpl
K8S_CRD_COMPONENT_SOURCE?=${K8S_HELM_CRD_RESSOURCES}/no-file-configured

##@ K8s - CRD targets

.PHONY: manifests
manifests: ${K8S_CRD_COMPONENT_SOURCE} ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.

${K8S_CRD_COMPONENT_SOURCE}: ${CRD_SRC_GO} ${CONTROLLER_GEN}
	@echo "Generate manifests..."
	@$(CONTROLLER_GEN) crd paths="./..." output:crd:artifacts:config=k8s/helm-crd/templates

.PHONY: crd-helm-generate-chart ## Generates the Helm CRD chart
crd-helm-generate-chart: validate-crd-chart validate-crd ${K8S_HELM_CRD_TARGET}/Chart.yaml

# this is phony because of it is easier this way than the makefile-single-run way
.PHONY: ${K8S_HELM_CRD_TARGET}/Chart.yaml
${K8S_HELM_CRD_TARGET}/Chart.yaml: ${K8S_RESOURCE_TEMP_FOLDER}
	@echo "Copying Helm CRD files..."
	@rm -drf ${K8S_HELM_CRD_TARGET}/templates
	@mkdir -p ${K8S_HELM_CRD_TARGET}/templates
	@cp -r ${K8S_HELM_CRD_RESSOURCES}/** ${K8S_HELM_CRD_TARGET}

	@echo "Generate Helm CRD chart..."
	@sed -i 's/name: artifact-crd-replaceme/name: ${ARTIFACT_CRD_ID}/' ${K8S_HELM_CRD_TARGET}/Chart.yaml
	@if [[ ${STAGE} == "development" ]]; then \
	  sed -i 's/appVersion: "0.0.0-replaceme"/appVersion: "${DEV_CRD_VERSION}"/' ${K8S_HELM_CRD_TARGET}/Chart.yaml; \
      sed -i 's/version: 0.0.0-replaceme/version: ${DEV_CRD_VERSION}/' ${K8S_HELM_CRD_TARGET}/Chart.yaml; \
	else \
	  sed -i 's/appVersion: "0.0.0-replaceme"/appVersion: "${VERSION}"/' ${K8S_HELM_CRD_TARGET}/Chart.yaml; \
      sed -i 's/version: 0.0.0-replaceme/version: ${VERSION}/' ${K8S_HELM_CRD_TARGET}/Chart.yaml; \
	fi

.PHONY: validate-crd-chart
validate-crd-chart:
	@if [ ! -f ${K8S_HELM_CRD_RESSOURCES}/Chart.yaml ] ; then \
       echo "Could not find CRD source Helm chart under \$${K8S_HELM_CRD_RESSOURCES}/Chart.yaml" ; \
       exit 23 ; \
    fi

.PHONY: validate-crd
validate-crd:
	@if [ ! -f ${K8S_CRD_COMPONENT_SOURCE} ] ; then \
       echo "Could not find CRD source. Did you point 'K8S_CRD_COMPONENT_SOURCE' to your source CRD file?" ; \
       exit 24 ; \
    fi

.PHONY: crd-helm-apply
crd-helm-apply: ${K8S_CRD_COMPONENT_SOURCE} ${BINARY_HELM} check-k8s-namespace-env-var crd-helm-generate-chart $(K8S_POST_GENERATE_TARGETS) ## Generates and installs the Helm CRD chart.
	@echo "Apply generated Helm CRD chart"
	@${BINARY_HELM} upgrade -i ${ARTIFACT_CRD_ID} ${K8S_HELM_CRD_TARGET} ${BINARY_HELM_ADDITIONAL_UPGR_ARGS} --namespace ${NAMESPACE}

.PHONY: crd-helm-delete
crd-helm-delete: ${BINARY_HELM} check-k8s-namespace-env-var ## Uninstalls the current Helm CRD chart.
	@echo "Uninstall Helm CRD chart"
	@${BINARY_HELM} uninstall ${ARTIFACT_CRD_ID} --namespace=${NAMESPACE} ${BINARY_HELM_ADDITIONAL_UNINST_ARGS} || true

.PHONY: crd-helm-package
crd-helm-package: crd-helm-delete-existing-tgz ${K8S_HELM_CRD_RELEASE_TGZ} ## Generates and packages the Helm CRD chart.

.PHONY: crd-helm-delete-existing-tgz
crd-helm-delete-existing-tgz: ## Remove an existing Helm CRD package.
	@rm -f ${K8S_HELM_CRD_RELEASE_TGZ}*

${K8S_HELM_CRD_RELEASE_TGZ}: ${BINARY_HELM} crd-helm-generate-chart $(K8S_POST_GENERATE_TARGETS) ## Generates and packages the Helm CRD chart.
	@echo "Package generated helm crd-chart"
	@${BINARY_HELM} package ${K8S_HELM_CRD_TARGET} -d ${K8S_HELM_CRD_TARGET} ${BINARY_HELM_ADDITIONAL_PACK_ARGS}

.PHONY: crd-helm-chart-import
crd-helm-chart-import: check-all-vars check-k8s-artifact-id crd-helm-generate-chart crd-helm-package ## Imports the currently available Helm CRD chart into the cluster-local registry.
	@if [[ ${STAGE} == "development" ]]; then \
		echo "Import ${K8S_HELM_CRD_DEV_RELEASE_TGZ} into K8s cluster ${K3CES_REGISTRY_URL_PREFIX}..."; \
		${BINARY_HELM} push ${K8S_HELM_CRD_DEV_RELEASE_TGZ} oci://${K3CES_REGISTRY_URL_PREFIX}/${K8S_HELM_ARTIFACT_NAMESPACE} ${BINARY_HELM_ADDITIONAL_PUSH_ARGS}; \
	else \
	  	echo "Import ${K8S_HELM_CRD_RELEASE_TGZ} into K8s cluster ${K3CES_REGISTRY_URL_PREFIX}..."; \
        ${BINARY_HELM} push ${K8S_HELM_CRD_RELEASE_TGZ} oci://${K3CES_REGISTRY_URL_PREFIX}/${K8S_HELM_ARTIFACT_NAMESPACE} ${BINARY_HELM_ADDITIONAL_PUSH_ARGS}; \
    fi
	@echo "Done."

.PHONY: crd-component-generate
crd-component-generate: ${K8S_RESOURCE_TEMP_FOLDER} ## Generate the CRD component YAML resource.
	@echo "Generating temporary K8s crd-component resource: ${K8S_RESOURCE_CRD_COMPONENT}"
	@if [[ ${STAGE} == "development" ]]; then \
		sed "s|NAMESPACE|$(K8S_HELM_ARTIFACT_NAMESPACE)|g" "${K8S_RESOURCE_COMPONENT_CR_TEMPLATE_YAML}" | sed "s|NAME|$(ARTIFACT_CRD_ID)|g"  | sed "s|VERSION|$(DEV_CRD_VERSION)|g" > "${K8S_RESOURCE_CRD_COMPONENT}"; \
	else \
		sed "s|NAMESPACE|$(K8S_HELM_ARTIFACT_NAMESPACE)|g" "${K8S_RESOURCE_COMPONENT_CR_TEMPLATE_YAML}" | sed "s|NAME|$(ARTIFACT_CRD_ID)|g"  | sed "s|VERSION|$(VERSION)|g" > "${K8S_RESOURCE_CRD_COMPONENT}"; \
	fi

.PHONY: crd-component-apply
crd-component-apply: check-k8s-namespace-env-var crd-helm-chart-import crd-component-generate $(K8S_POST_GENERATE_TARGETS) ## Applies the CRD component YAML resource to the actual defined context.
	@kubectl apply -f "${K8S_RESOURCE_CRD_COMPONENT}" --namespace="${NAMESPACE}"
	@echo "Done."

.PHONY: crd-component-delete
crd-component-delete: check-k8s-namespace-env-var crd-component-generate $(K8S_POST_GENERATE_TARGETS) ## Deletes the CRD component YAML resource from the actual defined context.
	@kubectl delete -f "${K8S_RESOURCE_CRD_COMPONENT}" --namespace="${NAMESPACE}" || true
	@echo "Done."
