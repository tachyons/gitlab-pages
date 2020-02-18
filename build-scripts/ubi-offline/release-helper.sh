#!/usr/bin/env bash

#
# This is a helper script for CNG UBI source release. This script:
#
#   - Creates a directory for the new release using the specified release tag.
#   - Duplicates the UBI image directories (Dockerfiles and assets) and
#     restructures and renames the files where needed.
#   - Replaces CI-specific Dockerfile arguments.
#   - Prepends the build stage Dockerfile instructions.
#   - Copies some of the assets from the previous release, including, LICENSE,
#     README.md, and build-scripts/+
#
# USAGE:
#
#   release-helper RELEASE_TAG RELEASE_PATH PREVIOUS_RELEASE [SOURCE]
#
#     RELEASE_TAG       GitLab release tag (e.g. v12.5.1-ubi8). The scripts
#                       creates a directory for the release tag (e.g. 12.5).
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
# NOTE:
#
#   This script requires GNU sed.
#
# CAVEATS:
#
#   This script does not set the version build arguments, such as `REGISTRY_VERSION`
#   in `gitlab-container-registry` or `GITALY_SERVER_VERSION` in `gitaly`.
#

set -euxo pipefail

SCRIPT_HOME="$( cd "${BASH_SOURCE[0]%/*}" > /dev/null 2>&1 && pwd )"

RELEASE_TAG="${1}"
RELEASE_PATH="${2}"
PREVIOUS_RELEASE="${3}"
SOURCE="${4:-.}"

DOCKERFILE_EXT='.ubi8'
GITLAB_REPOSITORY='${BASE_REGISTRY}/gitlab/gitlab/'

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
  local DOCKERFILE="${1}"
  local IMAGE_NAME="${2}"
  cat - "${DOCKERFILE}" > "${DOCKERFILE}.0" <<-EOF
ARG GITLAB_VERSION=${RELEASE_TAG}

ARG BASE_REGISTRY=registry.access.redhat.com
ARG BASE_IMAGE=ubi8/ubi
ARG BASE_TAG=8.1

ARG UBI_IMAGE=\${BASE_REGISTRY}/\${BASE_IMAGE}:\${BASE_TAG}

FROM \${UBI_IMAGE} AS builder

ARG NEXUS_SERVER
ARG VENDOR=gitlab
ARG PACKAGE_NAME=ubi8-build-dependencies-\${GITLAB_VERSION}.tar
ARG PACKAGE_URL=https://\${NEXUS_SERVER}/repository/dsop/\${VENDOR}/${IMAGE_NAME}/\${PACKAGE_NAME}

ADD build-scripts/ /build-scripts/

RUN /build-scripts/prepare.sh "\${PACKAGE_URL}"

EOF
  mv "${DOCKERFILE}.0" "${DOCKERFILE}"
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
    sed -i "s/^ARG ${_KEY}/ARG ${_KEY}=${LABELED_VERSIONS[$_KEY]}/g" "${DOCKERFILE}"
  done
}

addLicense() {
  local IMAGE_NAME="${1}"
  local IMAGE_ROOT="${2}"
  cp -n "${RELEASE_PATH}/${IMAGE_NAME}/${PREVIOUS_RELEASE}/LICENSE" "${IMAGE_ROOT}/"
}

addReadMe() {
  local IMAGE_NAME="${1}"
  local IMAGE_ROOT="${2}"
  cp -n "${RELEASE_PATH}/${IMAGE_NAME}/${PREVIOUS_RELEASE}/README.md" "${IMAGE_ROOT}/"
}

addBuildScripts() {
  local FULL_IMAGE_NAME="${1}"
  local IMAGE_NAME="${1%*-ee}";
  local IMAGE_TAG="${2}"
  local IMAGE_ROOT="${3}"
  cp -Rn "${RELEASE_PATH}/${IMAGE_NAME}/${PREVIOUS_RELEASE}/build-scripts" "${IMAGE_ROOT}/"
  chmod +x "${IMAGE_ROOT}/build-scripts"/*.sh
  sed -i "s/^TAG=.*/TAG=\$\{1:-${IMAGE_TAG}\}/g" "${IMAGE_ROOT}/build-scripts/build.sh"
  if [ -f "${RELEASE_PATH}/${IMAGE_NAME}/${PREVIOUS_RELEASE}/scripts/prebuild.sh" ]; then
    mkdir -p "${IMAGE_ROOT}/scripts"
    cp -n "${RELEASE_PATH}/${IMAGE_NAME}/${PREVIOUS_RELEASE}/scripts/prebuild.sh" "${IMAGE_ROOT}/scripts"
    sed -i "s/^GITLAB_VERSION=.*/GITLAB_VERSION=${RELEASE_TAG}/g" "${IMAGE_ROOT}/scripts/prebuild.sh"
    chmod +x "${IMAGE_ROOT}/scripts/prebuild.sh"
  fi
}

cleanupDirectory() {
  rm -rf "${IMAGE_ROOT}"/{patches,vendor,renderDockerfile,Dockerfile.erb,centos-8-base.repo}
}

releaseImage() {
  local IMAGE_NAME="${1%*-ee}"; local FULL_IMAGE_NAME="${1}"; shift
  local IMAGE_TAG="${RELEASE_TAG%.*}"
  IMAGE_TAG="${IMAGE_TAG#v*}"
  local IMAGE_ROOT="${RELEASE_PATH}/${IMAGE_NAME}/${IMAGE_TAG}"
  local DOCKERFILE="${IMAGE_ROOT}/Dockerfile"
  duplicateImageDir "${IMAGE_NAME}" "${IMAGE_ROOT}"
  replaceUbiImageArg "${DOCKERFILE}"
  prependBuildStage "${DOCKERFILE}" "${IMAGE_NAME}" $@
  replaceRubyImageArg "${DOCKERFILE}" "${IMAGE_TAG}"
  replaceRailsImageArg "${DOCKERFILE}" "${IMAGE_TAG}"
  replaceGitImageArg "${DOCKERFILE}" "${IMAGE_TAG}"
  replaceAddDependencies "${DOCKERFILE}"
  addBuildScripts "${FULL_IMAGE_NAME}" "${IMAGE_TAG}" "${IMAGE_ROOT}"
  addLicense "${IMAGE_NAME}" "${IMAGE_ROOT}"
  addReadMe "${IMAGE_NAME}" "${IMAGE_ROOT}"
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
releaseImage gitlab-unicorn-ee gitlab-python
releaseImage gitlab-task-runner-ee gitlab-python
releaseImage gitlab-sidekiq-ee
releaseImage gitlab-workhorse-ee
