#!/usr/bin/env bash

#
# This is a helper script for CNG UBI source release. This script:
#
#   - Fetches and verifies the sha256 of the release assets
#   - Duplicates the UBI image directories (Dockerfiles and assets) and
#     restructures and renames the files where needed.
#   - Replaces CI-specific Dockerfile arguments.
#   - Prepends the build stage Dockerfile instructions.
#
# USAGE:
#
#   release-helper RELEASE_TAG RELEASE_PATH [SOURCE]
#
#     RELEASE_TAG       GitLab release tag (e.g. v12.5.1-ubi8).
#
#     RELEASE_PATH      The release root directory. A directory for the new
#                       release will be created here.
#
#     SOURCE            The root directory of GitLab CNG repository. The current
#                       directory is used when not specified.
#
# NOTE:
#
#   This script requires GNU sed, gpg, and curl
#

set -euxo pipefail

SCRIPT_HOME="$( cd "${BASH_SOURCE[0]%/*}" > /dev/null 2>&1 && pwd )"

RELEASE_TAG="${1}"
RELEASE_PATH="${2}"
SOURCE="${3:-.}"

DOCKERFILE_EXT='.ubi8'
GITLAB_REPOSITORY='${BASE_REGISTRY}/gitlab/gitlab/'
ASSET_SHA_FILE="/tmp/deps-${RELEASE_TAG}.tar.sha256"
ASSET_PUB_KEY_ID='5c7738cc4840f93f6e9170ff5a0e20d5f9706778'

declare -A LABELED_VERSIONS=(
  [REGISTRY_VERSION]=
  [GITLAB_EXPORTER_VERSION]=
  [GITLAB_SHELL_VERSION]=
  [WORKHORSE_VERSION]=
  [GITALY_SERVER_VERSION]=
  [MAILROOM_VERSION]=
)

for _KEY in "${!LABELED_VERSIONS[@]}"; do
  LABELED_VERSIONS[$_KEY]=$(
    grep "${_KEY}" "${SCRIPT_HOME}"/../../ci_files/variables.yml \
    | cut -d ':' -f 2 \
    | sed -e 's/[ \"]//g'
  )
done

# TODO move sha calculation to a signed asset
if [ ! -f "${ASSET_SHA_FILE}" ]; then
  gpg --batch --keyserver "keyserver.ubuntu.com" --recv-keys $ASSET_PUB_KEY_ID

  rm -f "/tmp/deps-${RELEASE_TAG}.tar" "/tmp/deps-${RELEASE_TAG}.tar.asc"
  curl --create-dirs "https://gitlab-ubi.s3.us-east-2.amazonaws.com/ubi8-build-dependencies-${RELEASE_TAG}.tar" -o "/tmp/deps-${RELEASE_TAG}.tar"
  curl --create-dirs "https://gitlab-ubi.s3.us-east-2.amazonaws.com/ubi8-build-dependencies-${RELEASE_TAG}.tar.asc" -o "/tmp/deps-${RELEASE_TAG}.tar.asc"
  gpg --verify "/tmp/deps-${RELEASE_TAG}.tar.asc" "/tmp/deps-${RELEASE_TAG}.tar"
  sha256sum "/tmp/deps-${RELEASE_TAG}.tar" | awk '{print $1}' > "${ASSET_SHA_FILE}"
fi

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

prependBaseArgs() {
  local DOCKERFILE="${1}"
  local IMAGE_NAME="${2}"
  cat - "${DOCKERFILE}" > "${DOCKERFILE}.0" <<-EOF
ARG GITLAB_VERSION=${RELEASE_TAG}

ARG BASE_REGISTRY=nexus-docker-secure.levelup-nexus.svc.cluster.local:18082
ARG BASE_IMAGE=redhat/ubi/ubi8
ARG BASE_TAG=8.1

ARG UBI_IMAGE=\${BASE_REGISTRY}/\${BASE_IMAGE}:\${BASE_TAG}

EOF
  mv "${DOCKERFILE}.0" "${DOCKERFILE}"
}

prependBuildStage() {
  local DOCKERFILE="${1}"
  local IMAGE_NAME="${2}"
  if grep -sq 'ADD .*.tar.gz' "${DOCKERFILE}"; then
    cat - "${DOCKERFILE}" > "${DOCKERFILE}.0" <<-EOF
FROM \${UBI_IMAGE} AS builder

ARG GITLAB_VERSION
ARG PACKAGE_NAME=ubi8-build-dependencies-\${GITLAB_VERSION}.tar

COPY \${PACKAGE_NAME} /opt/
ADD build-scripts/ /build-scripts/

RUN /build-scripts/prepare.sh "/opt/\${PACKAGE_NAME}"
EOF
    mv "${DOCKERFILE}.0" "${DOCKERFILE}"
  fi
}

replaceUbiImageArg() {
  local DOCKERFILE="${1}"
  sed -i '/ARG UBI_IMAGE=.*/d' "${DOCKERFILE}"
}

replaceRubyImageArg() {
  local DOCKERFILE="${1}"
  local IMAGE_TAG="${2}"
  if grep -sq 'ARG RUBY_IMAGE=' "${DOCKERFILE}"; then
    sed -i '/ARG RUBY_IMAGE=.*/d' "${DOCKERFILE}"
    sed -i "/ARG UBI_IMAGE=.*/a ARG RUBY_IMAGE=${GITLAB_REPOSITORY//\//\\/}gitlab-ruby:${IMAGE_TAG}" "${DOCKERFILE}"
  fi
}

replaceRailsImageArg() {
  local DOCKERFILE="${1}"
  local IMAGE_TAG="${2}"
  if grep -sq 'ARG RAILS_IMAGE=' "${DOCKERFILE}"; then
    sed -i '/ARG RAILS_IMAGE=.*/d' "${DOCKERFILE}"
    sed -i "/ARG UBI_IMAGE=.*/a ARG RAILS_IMAGE=${GITLAB_REPOSITORY//\//\\/}gitlab-rails:${IMAGE_TAG}" "${DOCKERFILE}"
  fi
}

replaceGitImageArg() {
  local DOCKERFILE="${1}"; shift
  local IMAGE_TAG="${1}"; shift
  if grep -sq 'ARG GIT_IMAGE=' "${DOCKERFILE}"; then
    sed -i '/ARG GIT_IMAGE=.*/d' "${DOCKERFILE}"
    sed -i "/ARG UBI_IMAGE=.*/a ARG GIT_IMAGE=${GITLAB_REPOSITORY//\//\\/}git-base:${IMAGE_TAG}" "${DOCKERFILE}"
  fi
}

replaceAddDependencies() {
  local DOCKERFILE="${1}"
  sed -i '0,/ADD .*\.tar\.gz/ s/ADD .*\.tar\.gz/COPY --from=builder \/prepare\/dependencies/' "${DOCKERFILE}"
  sed -i '/ADD .*\.tar\.gz/d' "${DOCKERFILE}"
}

replaceLabeledVersions() {
  local DOCKERFILE="${1}"
  for _KEY in "${!LABELED_VERSIONS[@]}"; do
    sed -i "s/^ARG ${_KEY}.*/ARG ${_KEY}=${LABELED_VERSIONS[$_KEY]}/g" "${DOCKERFILE}"
  done
}

updateBuildScripts() {
  local IMAGE_TAG="${1}"
  local IMAGE_ROOT="${2}"
  if [ -f "${IMAGE_ROOT}/build-scripts/build.sh" ]; then
    sed -i "s/^TAG=.*/TAG=\$\{1:-${IMAGE_TAG}\}/g" "${IMAGE_ROOT}/build-scripts/build.sh"
  fi
}

updateJenkinsfile() {
  local IMAGE_TAG="${1}"
  local IMAGE_ROOT="${2}"
  sed -i "s/version: \".*\"/version: \"${IMAGE_TAG}\"/g" "${IMAGE_ROOT}/Jenkinsfile"
}

updateDownload() {
  local IMAGE_ROOT="${1}"
  local ASSET_SHA=$(cat "${ASSET_SHA_FILE}")

  if [ -f "${IMAGE_ROOT}/download.yaml" ]; then
    sed -i "s/ubi8-build-dependencies-.*.tar/ubi8-build-dependencies-${RELEASE_TAG}.tar/g" "${IMAGE_ROOT}/download.yaml"
    sed -i "s/value: \".*\"/value: \"${ASSET_SHA}\"/g" "${IMAGE_ROOT}/download.yaml"
  fi

  if [ -f "${IMAGE_ROOT}/download.json" ]; then
    sed -i "s/ubi8-build-dependencies-.*.tar/ubi8-build-dependencies-${RELEASE_TAG}.tar/g" "${IMAGE_ROOT}/download.json"
    sed -i "s/\"value\": \".*\"/\"value\": \"${ASSET_SHA}\"/g" "${IMAGE_ROOT}/download.json"
  fi
}

cleanupDirectory() {
  rm -rf "${IMAGE_ROOT}"/{patches,vendor,renderDockerfile,Dockerfile.erb,centos-8-base.repo}
}

releaseImage() {
  local IMAGE_NAME="${1%*-ee}"; local FULL_IMAGE_NAME="${1}"; shift
  local IMAGE_TAG="${RELEASE_TAG%-*}"
  IMAGE_TAG="${IMAGE_TAG#v*}"
  local IMAGE_ROOT="${RELEASE_PATH}/${IMAGE_NAME}"
  local DOCKERFILE="${IMAGE_ROOT}/Dockerfile"
  duplicateImageDir "${IMAGE_NAME}" "${IMAGE_ROOT}"
  replaceUbiImageArg "${DOCKERFILE}"
  prependBuildStage "${DOCKERFILE}" "${IMAGE_NAME}"
  prependBaseArgs "${DOCKERFILE}" "${IMAGE_NAME}"
  replaceRubyImageArg "${DOCKERFILE}" "${IMAGE_TAG}"
  replaceRailsImageArg "${DOCKERFILE}" "${IMAGE_TAG}"
  replaceGitImageArg "${DOCKERFILE}" "${IMAGE_TAG}"
  replaceAddDependencies "${DOCKERFILE}"
  updateBuildScripts "${IMAGE_TAG}" "${IMAGE_ROOT}"
  updateJenkinsfile "${IMAGE_TAG}" "${IMAGE_ROOT}"
  updateDownload "${IMAGE_ROOT}"
  replaceLabeledVersions "${DOCKERFILE}"
  cleanupDirectory
}

mkdir -p "${RELEASE_PATH}"

releaseImage kubectl
releaseImage git-base
releaseImage gitlab-ruby
releaseImage gitlab-container-registry
releaseImage gitlab-shell
releaseImage gitaly gitlab-shell
releaseImage gitlab-exporter
releaseImage gitlab-mailroom
releaseImage gitlab-rails-ee
releaseImage gitlab-webservice-ee
releaseImage gitlab-task-runner-ee
releaseImage gitlab-sidekiq-ee
releaseImage gitlab-workhorse-ee
