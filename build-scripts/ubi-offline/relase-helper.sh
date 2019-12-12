#!/bin/bash

#
# This is a helper script for CNG UBI source release. This script:
#
#   - Creates a directory for the new release using the specified release tag.
#   - Duplicates the UBI image directories (Dockerfiles and assets) and
#     restructures and renames the files where needed.
#   - Replaces CI-specific Dockerfile arguments.
#   - Prepends the build stage Dockerfile instructuions.
#   - Copies some assets from the previous release, including, LICENSE, README.md,
#     and build-scripts/
#
# USAGE:
#
#   release-helper RELEASE_TAG RELEASE_PATH PREVIOUS_RELEASE [SOURCE]
#
#     RELEASE_TAG       GitLab release tag (e.g. v12.5.1). The scripts creates
#                       a directory for the release tag (e.g. 12.5).
#
#     RELEASE_PATH      The release root directory. A directory for the new
#                       release will be created here. The previous release 
#                       directory will be looked up in here as well.
#
#     PREVIOUS_RELEASE  The directory name of the previous release (e.g. 12.4).
#                       Some assets will be copied from the previous release.
#
#     SOURCE            The root directory of GitLab CNG repository. The current
#                       directory is used when not specified.
#
# NOTE: This script requires GNU sed.

set -euxo pipefail

SCRIPT_HOME="$( cd "${BASH_SOURCE[0]%/*}" > /dev/null 2>&1 && pwd )"

RELEASE_TAG="${1}"
RELEASE_PATH="${2}"
PREVIOUS_RELEASE="${3}"
SOURCE="${4:-.}"

DOCKERFILE_EXT='.ubi8'
GITLAB_REPOSITORY='registry.access.redhat.com/gitlab/gitlab/'

duplicateImageDir() {
  local IMAGE_NAME="${1}"
  local IMAGE_ROOT="${2}"
  if [ ! -f "${SOURCE}/${IMAGE_NAME}/Dockerfile${DOCKERFILE_EXT}" ]; then
    echo "Skipping ${IMAGE_NAME}"
    return 0
  fi
  mkdir -p "${IMAGE_ROOT}"
  cp -R "${SOURCE}/${IMAGE_NAME}"/* "${IMAGE_ROOT}"
  rm -f "${IMAGE_ROOT}"/{Dockerfile,"Dockerfile.build${DOCKERFILE_EXT}"}
  mv "${IMAGE_ROOT}/Dockerfile${DOCKERFILE_EXT}" "${IMAGE_ROOT}/Dockerfile"
}

prependBuildStage() {
  local DOCKERFILE="${1}"; shift
  cat - "${DOCKERFILE}" > "${DOCKERFILE}.0" <<-EOF
ARG BASE_REGISTRY=registry.access.redhat.com
ARG BASE_IMAGE=ubi8/ubi
ARG BASE_TAG=8.0

ARG UBI_IMAGE=\${BASE_REGISTRY}/\${BASE_IMAGE}:\${BASE_TAG}

FROM \${UBI_IMAGE} AS builder

ARG NEXUS_SERVER
ARG VENDOR=gitlab
ARG GITLAB_VERSION=${RELEASE_TAG}
ARG PACKAGE_NAME=ubi8-build-dependencies-\${GITLAB_VERSION}.tar
ARG PACKAGE_URL=https://\${NEXUS_SERVER}/repository/dsop/\${VENDOR}/kubectl/\${PACKAGE_NAME}

ADD build-scripts/ /build-scripts/

RUN /build-scripts/prepare.sh "\${PACKAGE_URL}" ${@}

EOF
  mv "${DOCKERFILE}.0" "${DOCKERFILE}"
}

replaceUbiImageArg() {
  local DOCKERFILE="${1}"
  sed -i 's/ARG UBI_IMAGE=.*/ARG UBI_IMAGE/g' "${DOCKERFILE}"
}

replaceRubyImageArg() {
  local DOCKERFILE="${1}"
  local IMAGE_TAG="${2}"
  sed -i "s/ARG RUBY_IMAGE=.*/ARG RUBY_IMAGE=${GITLAB_REPOSITORY//\//\\/}gitlab-ruby:${IMAGE_TAG}/g" "${DOCKERFILE}"
}

replaceRailsImageArg() {
  local DOCKERFILE="${1}"
  local IMAGE_TAG="${2}"
  sed -i "s/ARG RAILS_IMAGE=.*/ARG RAILS_IMAGE=${GITLAB_REPOSITORY//\//\\/}gitlab-rails:${IMAGE_TAG}/g" "${DOCKERFILE}"
}

replaceAddDependencies() {
  local DOCKERFILE="${1}"
  sed -i '0,/ADD .*\.tar\.gz/ s/ADD .*\.tar\.gz/COPY --from=builder \/prepare\/dependencies/' "${DOCKERFILE}"
  sed -i '/ADD .*\.tar\.gz/d' "${DOCKERFILE}"
}

addLicense() {
  local IMAGE_NAME="${1}"
  local IMAGE_ROOT="${2}"
  cp "${RELEASE_PATH}/${PREVIOUS_RELEASE}/${IMAGE_NAME}/LICENSE" "${IMAGE_ROOT}/"
}

addReadMe() {
  local IMAGE_NAME="${1}"
  local IMAGE_ROOT="${2}"
  cp "${RELEASE_PATH}/${PREVIOUS_RELEASE}/${IMAGE_NAME}/README.md" "${IMAGE_ROOT}/"
}

addBuildScripts() {
  local FULL_IMAGE_NAME="${1}"
  local IMAGE_NAME="${1%*-ee}";
  local IMAGE_TAG="${2}"
  local IMAGE_ROOT="${3}"
  mkdir "${IMAGE_ROOT}/build-scripts"
  cp -R "${RELEASE_PATH}/${PREVIOUS_RELEASE}/${IMAGE_NAME}/build-scripts" "${IMAGE_ROOT}/build-scripts"
}

releaseImage() {
  local IMAGE_NAME="${1%*-ee}"; local FULL_IMAGE_NAME="${1}"; shift
  local IMAGE_TAG="${RELEASE_TAG%.*}"
  IMAGE_TAG="${IMAGE_TAG#v*}"
  local IMAGE_ROOT="${RELEASE_PATH}/${IMAGE_NAME}/${IMAGE_TAG}"
  local DOCKERFILE="${IMAGE_ROOT}/Dockerfile"
  duplicateImageDir "${IMAGE_NAME}" "${IMAGE_ROOT}"
  replaceUbiImageArg "${DOCKERFILE}"
  replaceRubyImageArg "${DOCKERFILE}" "${IMAGE_TAG}"
  replaceRailsImageArg "${DOCKERFILE}" "${IMAGE_TAG}"
  prependBuildStage "${DOCKERFILE}" "${FULL_IMAGE_NAME}" $@
  replaceAddDependencies "${DOCKERFILE}"
  addBuildScripts "${FULL_IMAGE_NAME}" "${IMAGE_TAG}" "${IMAGE_ROOT}" 
  addLicense "${IMAGE_NAME}" "${IMAGE_ROOT}"
  addReadMe "${IMAGE_NAME}" "${IMAGE_ROOT}"
}

# TODO: MUST BE DELETED
rm -rf "${RELEASE_PATH}"

mkdir -p "${RELEASE_PATH}"

releaseImage kubectl
releaseImage gitlab-unicorn-ee gitlab-python
