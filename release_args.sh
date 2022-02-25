#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

# this function will be sourced from release.sh and be called from release_functions.sh
update_versions_modify_files() {
  newReleaseVersion="${1}"
  kustomizationYAML=config/manager/kustomization.yaml

  yq "with(.images[] | select(.name == \"controller\") ; .newTag = \"${newReleaseVersion}\")" "${kustomizationYAML}" \
    > tmpfile \
    && mv tmpfile "${kustomizationYAML}"
}

update_versions_stage_modified_files() {
  kustomizationYAML=config/manager/kustomization.yaml

  git add "${kustomizationYAML}"
}
