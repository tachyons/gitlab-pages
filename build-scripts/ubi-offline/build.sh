#!/bin/bash

# NOTICE: This script requires `docker`.

set -euxo pipefail

TAG=${1:-latest}
REPOSITORY=${2:-}

DOCKERFILE_EXT=".ubi8"
TAG_EXT="-ubi8"

imageName() {
  printf "${REPOSITORY}${1}:${TAG}${TAG_EXT}"
}

buildImage() {
  IMAGE="${1}"
  CONTEXT="${IMAGE%*-ee}"
  { 
    docker build \
      -f "${CONTEXT}/Dockerfile${DOCKERFILE_EXT}" \
      -t "$(imageName ${IMAGE})" \
      ${DOCKER_OPTS:-} \
      "${CONTEXT}" | tee ${CONTEXT}.out 
  } || {
    echo "${CONTEXT}" >> failed.log
  }
}

# Cleanup log outputs from previous build

rm -f *.out failed.log

# Stage one

buildImage kubectl &
buildImage gitlab-ruby &
buildImage gitlab-container-registry &
buildImage gitlab-redis-ha &

wait

# Stage two

DOCKER_OPTS="--build-arg RUBY_IMAGE=$(imageName gitlab-ruby)"
buildImage git-base &
buildImage gitlab-exporter &
buildImage gitlab-mailroom &
buildImage gitlab-shell &
buildImage gitlab-rails-ee &
buildImage gitlab-workhorse-ee &

wait

# Stage three

DOCKER_OPTS="--build-arg GIT_IMAGE=$(imageName git-base)"
buildImage gitaly &

DOCKER_OPTS="--build-arg RAILS_IMAGE=$(imageName gitlab-rails-ee)"
buildImage gitlab-geo-logcursor &
buildImage gitlab-sidekiq-ee &
buildImage gitlab-task-runner-ee &
buildImage gitlab-unicorn-ee &

wait
