#!/bin/bash
# Switches the Harbor registry (secret ces-container-registries) and the dogu registry v3 (secret dogu-registry-v3)
# between staging and prod. Used to test v3 dogus released to staging. Called by the Makefile targets
# registry-staging, registry-prod and registry-env-status.
#
# Usage: registry-env.sh staging|prod|status
#
# Environment:
#   NAMESPACE              namespace of the CES
#   YQ                     yq v4 binary
#   STAGING_REGISTRY_HOST  host of the staging Harbor
#   STAGING_DCC_ENDPOINT   endpoint of the staging dogu registry v3
#   PROD_REGISTRY_HOST     host of the prod Harbor
#   PROD_DCC_ENDPOINT      endpoint of the prod dogu registry v3
#
# The credentials for the target environment are prompted on every switch. They are used for Harbor and the DCC.
set -o errexit
set -o nounset
set -o pipefail

: "${NAMESPACE:?NAMESPACE must be set}"
: "${YQ:?YQ must be set}"
: "${STAGING_REGISTRY_HOST:?STAGING_REGISTRY_HOST must be set}"
: "${STAGING_DCC_ENDPOINT:?STAGING_DCC_ENDPOINT must be set}"
: "${PROD_REGISTRY_HOST:?PROD_REGISTRY_HOST must be set}"
: "${PROD_DCC_ENDPOINT:?PROD_DCC_ENDPOINT must be set}"

REGISTRIES_SECRET="ces-container-registries"
DCC_SECRET="dogu-registry-v3"
export DCC_SECRET

kube() {
  kubectl --namespace "${NAMESPACE}" "$@"
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

read_credentials() {
  read -r -p "$1 username: " USERNAME || true
  read -r -s -p "$1 password: " PASSWORD || true
  echo
  [[ -n "${USERNAME}" && -n "${PASSWORD}" ]] || fail "username and password must not be empty"
  export USERNAME PASSWORD
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

# Removes the entries of host $1, with and without https:// scheme, from ces-container-registries.
remove_registry() {
  echo "Removing ${1} from ${REGISTRIES_SECRET}..."
  HOST="$1" update_registries_config 'del(.auths[strenv(HOST)], .auths["https://" + strenv(HOST)])'
}

# Writes the entry of host $1 with the prompted credentials to ces-container-registries. An entry with https:// scheme
# is replaced as well.
set_registry() {
  echo "Writing ${1} to ${REGISTRIES_SECRET}..."
  HOST="$1" update_registries_config 'del(.auths["https://" + strenv(HOST)]) | .auths[strenv(HOST)] = {
    "username": strenv(USERNAME),
    "password": strenv(PASSWORD),
    "auth": (strenv(USERNAME) + ":" + strenv(PASSWORD) | @base64)}'
}

# Points dogu-registry-v3 to endpoint $1 with the prompted credentials. Other keys of an existing secret stay untouched.
set_dcc() {
  if secret_exists "${DCC_SECRET}"; then
    echo "Pointing ${DCC_SECRET} to ${1}..."
    ENDPOINT="$1" "${YQ}" -n -o=json '[
      {"op": "add", "path": "/data/endpoint", "value": (strenv(ENDPOINT) | @base64)},
      {"op": "add", "path": "/data/username", "value": (strenv(USERNAME) | @base64)},
      {"op": "add", "path": "/data/password", "value": (strenv(PASSWORD) | @base64)}]' |
      patch_secret "${DCC_SECRET}"
  else
    echo "Creating ${DCC_SECRET} for ${1}..."
    ENDPOINT="$1" "${YQ}" -n -o=json '{"apiVersion": "v1", "kind": "Secret", "type": "Opaque",
      "metadata": {"name": strenv(DCC_SECRET)},
      "stringData": {
        "endpoint": strenv(ENDPOINT),
        "username": strenv(USERNAME),
        "password": strenv(PASSWORD),
        "urlschema": "default",
        "insecureSkipVerify": "false"}}' |
      create_secret
  fi
}

staging() {
  require_registries_secret
  read_credentials "Staging"

  set_registry "${STAGING_REGISTRY_HOST}"
  set_dcc "${STAGING_DCC_ENDPOINT}"
}

prod() {
  require_registries_secret
  read_credentials "Prod"

  remove_registry "${STAGING_REGISTRY_HOST}"
  set_registry "${PROD_REGISTRY_HOST}"
  set_dcc "${PROD_DCC_ENDPOINT}"
}

status() {
  echo "Namespace: ${NAMESPACE}"

  if secret_exists "${DCC_SECRET}"; then
    echo "DCC v3 endpoint: $(kube get secret "${DCC_SECRET}" -o jsonpath='{.data.endpoint}' | base64 -d)"
  else
    echo "DCC v3 endpoint: <secret ${DCC_SECRET} not found>"
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
