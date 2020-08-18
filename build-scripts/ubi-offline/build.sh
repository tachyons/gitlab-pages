#!/bin/bash

#
# Builds all UBI-based images in the right order.
#
# USAGE:
#
#   build.sh
#
# NOTE:
#
#   This script requires `docker`.
#

set -euxo pipefail

SCRIPT_HOME="$( cd "${BASH_SOURCE[0]%/*}" > /dev/null 2>&1 && pwd )"

TAG=${1:-master}
REPOSITORY=${2:-}
WORKSPACE="${SCRIPT_HOME}/build"
DOCKERFILE_EXT='.ubi8'
TAG_EXT='-ubi8'

mkdir -p "${WORKSPACE}"

qualifiedName() {
  printf "${REPOSITORY}${1}:${TAG}${TAG_EXT}"
}

buildImage() {
  IMAGE_NAME="${1}"
  IMAGE_DIR="${IMAGE_NAME%*-ee}"
  CONTEXT="${SCRIPT_HOME}/../../${IMAGE_DIR}"
  {
    docker build \
      -f "${CONTEXT}/Dockerfile${DOCKERFILE_EXT}" \
      -t "$(qualifiedName ${IMAGE_NAME})" \
      ${DOCKER_OPTS:-} \
      "${CONTEXT}" 2>&1 | tee "${WORKSPACE}/${IMAGE_DIR}.out"
  } || {
    echo "${IMAGE_DIR}" >> "${WORKSPACE}/failed.log"
  }
}

# Cleanup log outputs from previous build

rm -f "${WORKSPACE}"/*.out "${WORKSPACE}/failed.log"

# Stage one

buildImage kubectl &
buildImage gitlab-ruby &
buildImage gitlab-container-registry &

wait

# Stage two

DOCKER_OPTS="--build-arg RUBY_IMAGE=$(qualifiedName gitlab-ruby)"
buildImage git-base &
buildImage gitlab-exporter &
buildImage gitlab-mailroom &
buildImage gitlab-shell &
buildImage gitlab-rails-ee &
buildImage gitlab-workhorse-ee &

wait

# Stage three

DOCKER_OPTS="--build-arg GIT_IMAGE=$(qualifiedName git-base)"
buildImage gitaly &

DOCKER_OPTS="--build-arg RAILS_IMAGE=$(qualifiedName gitlab-rails-ee)"
buildImage gitlab-geo-logcursor &
buildImage gitlab-sidekiq-ee &
buildImage gitlab-task-runner-ee &
buildImage gitlab-webservice-ee &

wait
