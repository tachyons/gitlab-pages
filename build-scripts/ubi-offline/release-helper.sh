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
NEXUS_UBI_IMAGE='${BASE_REGISTRY}/redhat/ubi/ubi8:8.2'

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

fetchAssetsSHA() {
  local ASSET_NAME="${1}"
  local ASSET_SHA_FILE="/tmp/deps-${RELEASE_TAG}-${ASSET_NAME}.sha256"

  if [ ! -f "${ASSET_SHA_FILE}" ]; then
    rm -f "${ASSET_SHA_FILE}"
    curl --create-dirs "https://gitlab-ubi.s3.us-east-2.amazonaws.com/ubi8-build-dependencies-${RELEASE_TAG}/${ASSET_NAME}.sha256" -o "${ASSET_SHA_FILE}"
  fi
  cat "${ASSET_SHA_FILE}"
}

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
  local IMAGE_TAG="${2:-8.2}"
  local BASE_IMAGE_PATH="${3:-redhat/ubi/ubi8}"
  cat - "${DOCKERFILE}" > "${DOCKERFILE}.0" <<-EOF
ARG GITLAB_VERSION=${RELEASE_TAG}

ARG BASE_REGISTRY=nexus-docker-secure.levelup-nexus.svc.cluster.local:18082
ARG BASE_IMAGE=${BASE_IMAGE_PATH}
ARG BASE_TAG=${IMAGE_TAG}

EOF
  mv "${DOCKERFILE}.0" "${DOCKERFILE}"

  if grep -sq 'ARG UBI_IMAGE=' "${DOCKERFILE}"; then
    sed -i "s/^ARG UBI_IMAGE=.*/ARG UBI_IMAGE=\${BASE_REGISTRY}\/\${BASE_IMAGE}:\${BASE_TAG}/g" "${DOCKERFILE}"
  fi
}

replaceRubyImageArg() {
  local DOCKERFILE="${1}"
  local IMAGE_TAG="${2}"
  if grep -sq 'ARG RUBY_IMAGE=' "${DOCKERFILE}"; then
    sed -i "s/^ARG UBI_IMAGE.*/ARG UBI_IMAGE=${NEXUS_UBI_IMAGE//\//\\/}/g" "${DOCKERFILE}"
    sed -i "s/^ARG RUBY_IMAGE=.*/ARG RUBY_IMAGE=\${BASE_REGISTRY}\/\${BASE_IMAGE}:\${BASE_TAG}/g" "${DOCKERFILE}"
  fi
}

replaceRailsImageArg() {
  local DOCKERFILE="${1}"
  local IMAGE_TAG="${2}"
  if grep -sq 'ARG RAILS_IMAGE=' "${DOCKERFILE}"; then
    sed -i "s/^ARG UBI_IMAGE.*/ARG UBI_IMAGE=${NEXUS_UBI_IMAGE//\//\\/}/g" "${DOCKERFILE}"
    sed -i '/ARG RAILS_IMAGE=.*/d' "${DOCKERFILE}"
    if grep -sq 'ARG UBI_IMAGE=' "${DOCKERFILE}"; then
      sed -i "/ARG UBI_IMAGE=.*/a ARG RAILS_IMAGE=\${BASE_REGISTRY}/\${BASE_IMAGE}:\${BASE_TAG}" "${DOCKERFILE}"
    else
      sed -i "/ARG BASE_TAG=.*/a \\\nARG RAILS_IMAGE=\${BASE_REGISTRY}/\${BASE_IMAGE}:\${BASE_TAG}" "${DOCKERFILE}"
      sed -i "/ARG RAILS_IMAGE=.*/{n;d;}" "${DOCKERFILE}"
    fi
  fi
}

replaceGitImageArg() {
  local DOCKERFILE="${1}"; shift
  local IMAGE_TAG="${1}"; shift
  if grep -sq 'ARG GIT_IMAGE=' "${DOCKERFILE}"; then
    sed -i "s/^ARG UBI_IMAGE.*/ARG UBI_IMAGE=${NEXUS_UBI_IMAGE//\//\\/}/g" "${DOCKERFILE}"
    sed -i "s/^ARG GIT_IMAGE=.*/ARG GIT_IMAGE=\${BASE_REGISTRY}\/\${BASE_IMAGE}:\${BASE_TAG}/g" "${DOCKERFILE}"
  fi
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
  if [ -f "${IMAGE_ROOT}/Jenkinsfile" ]; then
    sed -i "s/version: \"[0-9]\+\.[0-9]\+\.[0-9]\+/version: \"${IMAGE_TAG}/g" "${IMAGE_ROOT}/Jenkinsfile"
  fi
}

appendResource() {
  local RESOURCE_NAME="${1}"
  local IMAGE_ROOT="${2}"
  local ASSET_SHA
  ASSET_SHA=$(fetchAssetsSHA "${RESOURCE_NAME}")

  cat <<EOF >> "${IMAGE_ROOT}/download.yaml"
  - url: "http://gitlab-ubi.s3.amazonaws.com/ubi8-build-dependencies-${RELEASE_TAG}/${RESOURCE_NAME}"
    filename: "${RESOURCE_NAME}"
    validation:
      type: "sha256"
      value: "${ASSET_SHA}"
EOF
}

createDownload() {
  local IMAGE_ROOT="${1}"

  echo "resources:" > "${IMAGE_ROOT}/download.yaml"

  local resources=($(sed -rn 's/^ADD (.*.tar.gz).*$/\1/p' "${IMAGE_ROOT}/Dockerfile"))
  for resource in "${resources[@]}"
  do
     appendResource "${resource}" "${IMAGE_ROOT}"
  done

  # standardize on the yaml, remove any json download file
  rm -rf "${IMAGE_ROOT}/download.json"
}

cleanupDirectory() {
  rm -rf "${IMAGE_ROOT}"/{patches,vendor,renderDockerfile,Dockerfile.erb,centos-8-base.repo}

  # remove old prepare scripts
  rm -f "${IMAGE_ROOT}/build-scripts/prepare.sh"
}

releaseImage() {
  local IMAGE_NAME="${1%*-ee}"; local FULL_IMAGE_NAME="${1}"; shift
  local BASE_IMAGE="${1:-}"
  local IMAGE_TAG="${RELEASE_TAG%-*}"
  IMAGE_TAG="${IMAGE_TAG#v*}"
  local IMAGE_ROOT="${RELEASE_PATH}/${IMAGE_NAME}"
  local DOCKERFILE="${IMAGE_ROOT}/Dockerfile"
  local BASE_TAG=""
  local BASE_IMAGE_PATH=""

  if [ ! -z $BASE_IMAGE ]; then
    BASE_TAG=${IMAGE_TAG}
    BASE_IMAGE_PATH="gitlab/gitlab/${BASE_IMAGE}"
  fi

  duplicateImageDir "${IMAGE_NAME}" "${IMAGE_ROOT}"
  prependBaseArgs "${DOCKERFILE}" "${BASE_TAG}" "${BASE_IMAGE_PATH}"
  replaceRubyImageArg "${DOCKERFILE}" "${IMAGE_TAG}"
  replaceRailsImageArg "${DOCKERFILE}" "${IMAGE_TAG}"
  replaceGitImageArg "${DOCKERFILE}" "${IMAGE_TAG}"
  updateBuildScripts "${IMAGE_TAG}" "${IMAGE_ROOT}"
  updateJenkinsfile "${IMAGE_TAG}" "${IMAGE_ROOT}"
  createDownload "${IMAGE_ROOT}"
  replaceLabeledVersions "${DOCKERFILE}"
  cleanupDirectory
}

mkdir -p "${RELEASE_PATH}"

releaseImage kubectl
releaseImage git-base "gitlab-ruby"
releaseImage gitlab-ruby
releaseImage gitlab-container-registry
releaseImage gitlab-shell "gitlab-ruby"
releaseImage gitaly "git-base"
releaseImage gitlab-exporter "gitlab-ruby"
releaseImage gitlab-mailroom "gitlab-ruby"
releaseImage gitlab-rails-ee "gitlab-ruby"
releaseImage gitlab-webservice-ee "gitlab-rails"
releaseImage gitlab-task-runner-ee "gitlab-rails"
releaseImage gitlab-sidekiq-ee "gitlab-rails"
releaseImage gitlab-workhorse-ee "gitlab-ruby"
