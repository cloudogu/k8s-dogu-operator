# This file is optional and can be used to set personal information without committing them to the repository.
MY_ENV_FILE ?= $(WORKDIR)/.env
ifneq (,$(wildcard $(MY_ENV_FILE)))
    include .env
endif

## Variables

# The cluster root variable is used to the build images to the cluster. It can be defined in a .myenv file.
K8S_CLUSTER_ROOT ?=
# The productive tag of the image
IMAGE ?=

# Variables for the temporary yaml files. These are used as template to generate a development resource containing
# the current namespace and the dev image.
K8S_RESOURCE_TEMP_FOLDER ?= $(TARGET_DIR)/make/k8s
K8S_RESOURCE_TEMP_YAML ?= $(K8S_RESOURCE_TEMP_FOLDER)/$(ARTIFACT_ID).yaml

# The current namespace is extracted from the current context.
K8S_CURRENT_NAMESPACE=$(shell kubectl config view --minify -o jsonpath='{..namespace}')

##@ K8s - Variables

.PHONY: check-all-vars
check-all-vars: check-k8s-cluster-root-env-var check-k8s-image-env-var check-k8s-artifact-id

.PHONY: check-k8s-cluster-root-env-var
check-k8s-cluster-root-env-var:
	@$(call check_defined, K8S_CLUSTER_ROOT, root path of your k3ces)

.PHONY: check-k8s-image-env-var
check-k8s-image-env-var:
	@$(call check_defined, IMAGE, docker image tag)

.PHONY: check-k8s-artifact-id
check-k8s-artifact-id:
	@$(call check_defined, ARTIFACT_ID, app/dogu name)

##@ K8s - Resources

${K8S_RESOURCE_TEMP_FOLDER}:
	@mkdir -p $@

.PHONY: k8s-delete
k8s-delete: k8s-generate $(K8S_POST_GENERATE_TARGETS) ## Deletes all dogu related resources from the K8s cluster.
	@echo "Delete old dogu resources..."
	@kubectl delete -f $(K8S_RESOURCE_TEMP_YAML) --wait=false --ignore-not-found=true

# The additional targets executed after the generate target, executed before each apply and delete. The generate target
# produces a temporary yaml. This yaml is accessible via K8S_RESOURCE_TEMP_YAML an can be changed before the apply/delete.
K8S_POST_GENERATE_TARGETS ?=
# The additional targets executed before the generate target, executed before each apply and delete.
K8S_PRE_GENERATE_TARGETS ?= k8s-create-temporary-resource
.PHONY: k8s-generate
k8s-generate: $(K8S_RESOURCE_TEMP_FOLDER) $(K8S_PRE_GENERATE_TARGETS) ## Generates the final resource yaml.
	@echo "Applying general transformations..."
	@sed -i "s/'{{ .Namespace }}'/$(K8S_CURRENT_NAMESPACE)/" $(K8S_RESOURCE_TEMP_YAML)
	@yq -i e "(select(.kind == \"Deployment\").spec.template.spec.containers[]|select(.image == \"*$(ARTIFACT_ID)*\").image)=\"$(IMAGE)\"" $(K8S_RESOURCE_TEMP_YAML)
	@echo "Done."

.PHONY: k8s-apply
k8s-apply: k8s-generate $(K8S_POST_GENERATE_TARGETS) ## Applies all dogu related resources from the K8s cluster.
	@echo "Apply new dogu resources..."
	@kubectl apply -f $(K8S_RESOURCE_TEMP_YAML)

##@ K8s - Docker

.PHONY: docker-build
docker-build: check-k8s-image-env-var ## Builds the docker image of the k8s app.
	@echo "Building docker image of dogu..."
	DOCKER_BUILDKIT=1 docker build . -t $(IMAGE)

${K8S_CLUSTER_ROOT}/${ARTIFACT_ID}.tar: check-k8s-artifact-id docker-build	## Saves the current image into a file into the K8s root path to be available on all nodes.
	@echo "Saving image into k3ces as $(ARTIFACT_ID).tar..."
	@docker save $(IMAGE) -o $(K8S_CLUSTER_ROOT)/$(ARTIFACT_ID).tar
	@echo "Done."

.PHONY: image-import
image-import: check-all-vars $(K8S_CLUSTER_ROOT)/$(ARTIFACT_ID).tar ## Imports the currently available image into the K8s cluster for all nodes.
	@echo "Import $(K8S_CLUSTER_ROOT)/$(ARTIFACT_ID).tar into all K8s nodes..."
	@cd $(K8S_CLUSTER_ROOT) && \
		for node in $$(vagrant status --machine-readable | grep "state,running" | awk -F',' '{print $$2}'); \
		do  \
			echo "...$${node}"; \
			vagrant ssh $${node} -- -t "sudo k3s ctr images import /vagrant/${ARTIFACT_ID}.tar"; \
		done;
	@echo "Done."
	rm $(K8S_CLUSTER_ROOT)/$(ARTIFACT_ID).tar

## Functions

# Check that given variables are set and all have non-empty values,
# die with an error otherwise.
#
# Params:
#   1. Variable name(s) to test.
#   2. (optional) Error message to print.
check_defined = \
    $(strip $(foreach 1,$1, \
        $(call __check_defined,$1,$(strip $(value 2)))))
__check_defined = \
    $(if $(value $1),, \
      $(error Undefined $1$(if $2, ($2))))
