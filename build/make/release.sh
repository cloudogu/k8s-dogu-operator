#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

source "$(pwd)/build/make/release_functions.sh"

TYPE="${1}"

echo "=====Starting Release process====="

if [ "${TYPE}" == "dogu" ];then
  CURRENT_TOOL_VERSION=$(get_current_version_by_dogu_json)
else
  CURRENT_TOOL_VERSION=$(get_current_version_by_makefile)
fi

NEW_RELEASE_VERSION="$(read_new_version)"

validate_new_version "${NEW_RELEASE_VERSION}"
start_git_flow_release "${NEW_RELEASE_VERSION}"
update_versions "${NEW_RELEASE_VERSION}"
update_changelog "${NEW_RELEASE_VERSION}"
show_diff
finish_release_and_push "${CURRENT_TOOL_VERSION}" "${NEW_RELEASE_VERSION}"

echo "=====Finished Release process====="
