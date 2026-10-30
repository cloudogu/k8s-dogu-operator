#!/bin/bash
# Switches the Harbor registry (secret ces-container-registries) and the dogu registry v3 (secret dogu-registry-v3)
# between staging and prod. Used to test v3 dogus released to staging. Called by the Makefile targets
# registry-staging, registry-prod and registry-env-status.
#
# Usage: registry-env.sh staging|prod|status
#
# Environment:
#   NAMESPACE              namespace of the CES
#   KUBE_CONTEXT           kube context to use (default: current context)
#   YQ                     yq v4 binary
#   STAGING_REGISTRY_HOST  host of the staging Harbor
#   STAGING_DCC_ENDPOINT   endpoint of the staging dogu registry v3
#   STAGING_USERNAME       staging credentials for Harbor and DCC, prompted if empty
#   STAGING_PASSWORD
set -o errexit
set -o nounset
set -o pipefail

: "${NAMESPACE:?NAMESPACE must be set}"
: "${YQ:?YQ must be set}"
: "${STAGING_REGISTRY_HOST:?STAGING_REGISTRY_HOST must be set}"
: "${STAGING_DCC_ENDPOINT:?STAGING_DCC_ENDPOINT must be set}"
export STAGING_REGISTRY_HOST STAGING_DCC_ENDPOINT

REGISTRIES_SECRET="ces-container-registries"
DCC_SECRET="dogu-registry-v3"
DCC_BACKUP_SECRET="dogu-registry-v3-prod-backup"
# Marks a backup of a dogu-registry-v3 secret that did not exist before switching to staging.
ABSENT_ANNOTATION="k8s.cloudogu.com/registry-env-secret-absent"
export DCC_SECRET DCC_BACKUP_SECRET ABSENT_ANNOTATION

kube() {
  kubectl ${KUBE_CONTEXT:+--context="${KUBE_CONTEXT}"} --namespace "${NAMESPACE}" "$@"
}

fail() {
  echo "ERROR: $*" >&2
  exit 1
}

secret_exists() {
  kube get secret "$1" >/dev/null 2>&1
}

# Secrets are passed via stdin so that credentials never show up in process arguments.
create_secret() {
  kube create -f - >/dev/null
}

# Applies the JSON patch on stdin to the secret $1. Unlike kubectl replace, a patch keeps the
# last-applied-configuration annotation untouched.
patch_secret() {
  kube patch secret "$1" --type=json --patch-file=/dev/stdin >/dev/null
}

require_registries_secret() {
  secret_exists "${REGISTRIES_SECRET}" ||
    fail "secret ${REGISTRIES_SECRET} not found in namespace ${NAMESPACE}. Wrong cluster or namespace?"
}

registries_config() {
  kube get secret "${REGISTRIES_SECRET}" -o json | "${YQ}" -p=json -o=yaml '.data[".dockerconfigjson"] | @base64d'
}

# Applies the yq expression $1 to the dockerconfigjson of ces-container-registries.
update_registries_config() {
  local config
  config="$(registries_config | "${YQ}" -p=json -o=json -I=0 "$1")"
  CONFIG="${config}" "${YQ}" -n -o=json \
    '[{"op": "replace", "path": "/data/.dockerconfigjson", "value": (strenv(CONFIG) | @base64)}]' |
    patch_secret "${REGISTRIES_SECRET}"
}

read_staging_credentials() {
  if [[ -z "${STAGING_USERNAME:-}" ]]; then
    read -r -p "Staging username: " STAGING_USERNAME || true
  fi
  if [[ -z "${STAGING_PASSWORD:-}" ]]; then
    read -r -s -p "Staging password: " STAGING_PASSWORD || true
    echo
  fi
  [[ -n "${STAGING_USERNAME}" && -n "${STAGING_PASSWORD}" ]] || fail "staging username and password must not be empty"
  export STAGING_USERNAME STAGING_PASSWORD
}

backup_dcc_secret() {
  if secret_exists "${DCC_BACKUP_SECRET}"; then
    echo "Backup ${DCC_BACKUP_SECRET} already exists, keeping it."
  elif secret_exists "${DCC_SECRET}"; then
    echo "Backing up ${DCC_SECRET} to ${DCC_BACKUP_SECRET}..."
    kube get secret "${DCC_SECRET}" -o json |
      "${YQ}" -p=json -o=json '{"apiVersion": "v1", "kind": "Secret", "type": .type,
        "metadata": {"name": strenv(DCC_BACKUP_SECRET)}, "data": .data}' |
      create_secret
  else
    echo "${DCC_SECRET} does not exist, recording its absence in ${DCC_BACKUP_SECRET}..."
    "${YQ}" -n -o=json '{"apiVersion": "v1", "kind": "Secret", "type": "Opaque",
      "metadata": {"name": strenv(DCC_BACKUP_SECRET), "annotations": {strenv(ABSENT_ANNOTATION): "true"}}}' |
      create_secret
  fi
}

switch_dcc_to_staging() {
  if secret_exists "${DCC_SECRET}"; then
    echo "Pointing ${DCC_SECRET} to ${STAGING_DCC_ENDPOINT}..."
    "${YQ}" -n -o=json '[
      {"op": "add", "path": "/data/endpoint", "value": (strenv(STAGING_DCC_ENDPOINT) | @base64)},
      {"op": "add", "path": "/data/username", "value": (strenv(STAGING_USERNAME) | @base64)},
      {"op": "add", "path": "/data/password", "value": (strenv(STAGING_PASSWORD) | @base64)}]' |
      patch_secret "${DCC_SECRET}"
  else
    echo "Creating ${DCC_SECRET} for ${STAGING_DCC_ENDPOINT}..."
    "${YQ}" -n -o=json '{"apiVersion": "v1", "kind": "Secret", "type": "Opaque",
      "metadata": {"name": strenv(DCC_SECRET)},
      "stringData": {
        "endpoint": strenv(STAGING_DCC_ENDPOINT),
        "username": strenv(STAGING_USERNAME),
        "password": strenv(STAGING_PASSWORD),
        "urlschema": "default",
        "insecureSkipVerify": "false"}}' |
      create_secret
  fi
}

# Restores only the data of dogu-registry-v3, so its labels and annotations stay untouched.
restore_dcc_secret() {
  if ! secret_exists "${DCC_BACKUP_SECRET}"; then
    echo "No backup ${DCC_BACKUP_SECRET} found (already on prod?), leaving ${DCC_SECRET} untouched."
    return
  fi

  local backup absent
  backup="$(kube get secret "${DCC_BACKUP_SECRET}" -o json)"
  absent="$("${YQ}" -p=json -o=yaml '.metadata.annotations[strenv(ABSENT_ANNOTATION)]' <<<"${backup}")"
  if [[ "${absent}" == "true" ]]; then
    echo "${DCC_SECRET} did not exist before switching to staging, deleting it..."
    kube delete secret "${DCC_SECRET}" --ignore-not-found >/dev/null
  elif secret_exists "${DCC_SECRET}"; then
    echo "Restoring ${DCC_SECRET} from ${DCC_BACKUP_SECRET}..."
    "${YQ}" -p=json -o=json '[{"op": "replace", "path": "/data", "value": .data}]' <<<"${backup}" |
      patch_secret "${DCC_SECRET}"
  else
    echo "Recreating ${DCC_SECRET} from ${DCC_BACKUP_SECRET}..."
    "${YQ}" -p=json -o=json '.metadata = {"name": strenv(DCC_SECRET)}' <<<"${backup}" | create_secret
  fi
  kube delete secret "${DCC_BACKUP_SECRET}" >/dev/null
}

staging() {
  require_registries_secret
  read_staging_credentials

  echo "Adding ${STAGING_REGISTRY_HOST} to ${REGISTRIES_SECRET}..."
  update_registries_config '.auths[strenv(STAGING_REGISTRY_HOST)] = {
    "username": strenv(STAGING_USERNAME),
    "password": strenv(STAGING_PASSWORD),
    "auth": (strenv(STAGING_USERNAME) + ":" + strenv(STAGING_PASSWORD) | @base64)}'

  backup_dcc_secret
  switch_dcc_to_staging
}

prod() {
  require_registries_secret

  local has_staging_registry
  has_staging_registry="$(registries_config | "${YQ}" -p=json -o=yaml '.auths | has(strenv(STAGING_REGISTRY_HOST))')"
  if [[ "${has_staging_registry}" == "true" ]]; then
    echo "Removing ${STAGING_REGISTRY_HOST} from ${REGISTRIES_SECRET}..."
    update_registries_config 'del(.auths[strenv(STAGING_REGISTRY_HOST)])'
  else
    echo "${STAGING_REGISTRY_HOST} is not part of ${REGISTRIES_SECRET}, nothing to remove."
  fi

  restore_dcc_secret
}

status() {
  echo "Namespace: ${NAMESPACE}"

  if secret_exists "${DCC_SECRET}"; then
    echo "DCC v3 endpoint: $(kube get secret "${DCC_SECRET}" -o jsonpath='{.data.endpoint}' | base64 -d)"
  else
    echo "DCC v3 endpoint: <secret ${DCC_SECRET} not found>"
  fi
  if secret_exists "${DCC_BACKUP_SECRET}"; then
    echo "Prod backup:     ${DCC_BACKUP_SECRET} exists"
  else
    echo "Prod backup:     none"
  fi

  echo "Registries in ${REGISTRIES_SECRET}:"
  if secret_exists "${REGISTRIES_SECRET}"; then
    registries_config | "${YQ}" -p=json -o=yaml '.auths | to_entries | .[] |
      "  " + .key + " (user: " + (.value.username // (.value.auth | @base64d | split(":") | .[0])) + ")"'
  else
    echo "  <secret ${REGISTRIES_SECRET} not found>"
  fi
}

case "${1:-}" in
  staging) staging ;;
  prod) prod ;;
  status) status ;;
  *) fail "usage: $0 staging|prod|status" ;;
esac
