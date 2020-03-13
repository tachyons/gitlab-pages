#!/bin/bash
declare -a nightly_builds=( gitlab-rails-ee gitlab-rails-ce gitlab-unicorn-ce gitlab-unicorn-ee gitaly gitlab-shell gitlab-sidekiq-ee gitlab-sidekiq-ce gitlab-workhorse-ce gitlab-workhorse-ee )

function _containsElement () {
  local e match="$1"
  shift
  for e; do [[ "$e" == "$match" ]] && return 0; done
  return 1
}

function is_nightly(){
  [ -n "$NIGHTLY" ] && $(_containsElement $CI_JOB_NAME ${nightly_builds[@]})
}

function is_master(){
  [ "$CI_COMMIT_REF_NAME" == "master" ]
}

function is_stable(){
  [[ "$CI_COMMIT_REF_NAME" =~ ^[0-9]+-[0-9]+-stable(-ee)?$ ]]
}

function force_build(){
  [ "${FORCE_IMAGE_BUILDS}" == "true" ]
}

function should_compile_assets() {
  [ "${COMPILE_ASSETS}" == "true" ]
}

function fetch_assets(){
  [ -z "${ASSETS_IMAGE}" ] && return 1
  should_compile_assets && return 0

  if needs_build; then
    while ! docker pull "${ASSETS_IMAGE}"; do
      echo "${ASSETS_IMAGE} not available yet. Sleeping for 30 seconds";
      sleep 30;
    done
  fi
}

function needs_build(){
  force_build || is_nightly || ! $(docker pull "$CI_REGISTRY_IMAGE/${CI_JOB_NAME#build:*}:$CONTAINER_VERSION${IMAGE_TAG_EXT}" > /dev/null);
}

function build_if_needed(){
  pushd $(get_trimmed_job_name)

  if [ -x renderDockerfile ]; then
    ./renderDockerfile
  fi

  if needs_build; then
    if [ ! -f "Dockerfile${DOCKERFILE_EXT}" ]; then
      echo "Skipping $(get_trimmed_job_name): Dockerfile${DOCKERFILE_EXT} does not exist."
      popd
      return 0
    fi

    export BUILDING_IMAGE="true"
    if [ -n "$BASE_IMAGE" ]; then
      docker pull $BASE_IMAGE
    fi

    DOCKER_ARGS=( "$@" )

    # Bring in shared scripts
    cp -r ../shared/ shared/

    # Skip the build cache if $DISABLE_DOCKER_BUILD_CACHE is set to any value
    if [ -z ${DISABLE_DOCKER_BUILD_CACHE+x} ]; then
      CACHE_IMAGE="$CI_REGISTRY_IMAGE/${CI_JOB_NAME#build:*}:$CI_COMMIT_REF_SLUG${IMAGE_TAG_EXT}"
      if ! $(docker pull $CACHE_IMAGE > /dev/null); then
        CACHE_IMAGE="$CI_REGISTRY_IMAGE/${CI_JOB_NAME#build:*}:latest${IMAGE_TAG_EXT}"
        docker pull $CACHE_IMAGE || true
      fi

      DOCKER_ARGS+=(--cache-from $CACHE_IMAGE)
    fi

    # Add build image argument for UBI build stage
    if [ "${UBI_BUILD_IMAGE}" = 'true' ]; then
      [ -z "${BUILD_IMAGE}" ] && export BUILD_IMAGE="${CI_REGISTRY_IMAGE}/gitlab-ubi-builder:latest-ubi8"
      DOCKER_ARGS+=(--build-arg BUILD_IMAGE="${BUILD_IMAGE}")
    fi

    docker build --build-arg CI_REGISTRY_IMAGE=$CI_REGISTRY_IMAGE -t "$CI_REGISTRY_IMAGE/${CI_JOB_NAME#build:*}:$CONTAINER_VERSION${IMAGE_TAG_EXT}" "${DOCKER_ARGS[@]}" -f Dockerfile${DOCKERFILE_EXT} ${DOCKER_BUILD_CONTEXT:-.}

    # Push new image unless it is a UBI build image
    if [ ! "${UBI_BUILD_IMAGE}" = 'true' ]; then
      docker push "$CI_REGISTRY_IMAGE/${CI_JOB_NAME#build:*}:$CONTAINER_VERSION${IMAGE_TAG_EXT}"

      # Create a tag based on Branch/Tag name for easy reference
      tag_and_push $CI_COMMIT_REF_SLUG${IMAGE_TAG_EXT}
    fi
  fi

  popd

  # Record image repository and tag unless it is a UBI build image
  if [ ! "${UBI_BUILD_IMAGE}" = 'true' ]; then
    echo "${CI_JOB_NAME#build:*}:$CONTAINER_VERSION${IMAGE_TAG_EXT}" > "artifacts/images/${CI_JOB_NAME#build:*}.txt"
  fi
}

function tag_and_push(){
  # Tag and push unless it is a UBI build image
  if [ ! "${UBI_BUILD_IMAGE}" = 'true' -a -f "$(get_trimmed_job_name)/Dockerfile${DOCKERFILE_EXT}" ]; then
    docker tag "$CI_REGISTRY_IMAGE/${CI_JOB_NAME#build:*}:$CONTAINER_VERSION${IMAGE_TAG_EXT}" "$CI_REGISTRY_IMAGE/${CI_JOB_NAME#build:*}:$1"
    docker push "$CI_REGISTRY_IMAGE/${CI_JOB_NAME#build:*}:$1"
  fi
}

function push_latest(){
  tag_and_push "latest${IMAGE_TAG_EXT}"
}

function get_version(){
  git ls-tree HEAD -- $1 | awk '{ print $3 }'
}

function get_target_version(){
  get_version $(get_trimmed_job_name)
}

function get_trimmed_job_name(){
  trim_edition ${CI_JOB_NAME#build:*}
}

function is_tag(){
  [ -n "${CI_COMMIT_TAG}" ] || [ -n "${GITLAB_TAG}" ]
}

function trim_edition(){
  echo $1 | sed -e "s/-.e\(-ubi8\)\?$/\1/"
}

function trim_tag(){
  echo $(trim_edition $1) | sed -e "s/^v//"
}

function push_if_master_or_stable_or_tag(){

  # For tag pipelines, nothing needs to be done on gitlab.com project. Images
  # will be built, and copied to .com registry as part of the release. However,
  # this check is done here intentionally, and not at build time (which
  # involves pushing CONTAINER_VERSION, CI_COMMIT_REF_SLUG tags also) because
  # we may not be syncing build images, but only the user facing images.
  if [ "$CI_REGISTRY" == "registry.gitlab.com" ] && [ -n "$CI_COMMIT_TAG" ]; then
    return
  fi

  if [ ! -f "$(get_trimmed_job_name)/Dockerfile${DOCKERFILE_EXT}" ]; then
    echo "Skipping $(get_trimmed_job_name): Dockerfile${DOCKERFILE_EXT} does not exist."
    return 0
  fi

  if is_master || is_stable || is_tag; then
    if [ -z "$1" ] || [ "$1" == "master" ]; then
      push_latest
    else
      local edition="$1"
      if is_tag; then
        edition=$(trim_edition $edition)
      fi
      tag_and_push $edition
      echo "${CI_JOB_NAME#build:*}:$edition" > "artifacts/images/${CI_JOB_NAME#build:*}.txt"
    fi
  fi
}

copy_assets() {
  if [ "${UBI_BUILD_IMAGE}" = 'true' ]; then
    ASSETS_DIR="artifacts/ubi/${CI_JOB_NAME#build:*}"
    mkdir -p "${ASSETS_DIR}"
    docker create --name assets "${CI_REGISTRY_IMAGE}/${CI_JOB_NAME#build:*}:${CONTAINER_VERSION}${IMAGE_TAG_EXT}"
    docker cp assets:/assets "${ASSETS_DIR}"
    docker rm assets
    tar -czf "${ASSETS_DIR}.tar.gz" -C "${ASSETS_DIR}/assets" .
    rm -rf "${ASSETS_DIR}"
  fi
}

use_assets() {
  if [ "${UBI_PIPELINE}" = 'true' -a -f "artifacts/ubi/${CI_JOB_NAME#build:*}.tar.gz" ]; then
    target="${CI_JOB_NAME#build:*}"
    cp -R "artifacts/ubi/${target}.tar.gz" "${target%*-ee}/${target}.tar.gz"
  fi
}

import_assets() {
  if [ "${UBI_PIPELINE}" = 'true' ]; then
    cp $@ $(get_trimmed_job_name)/
  fi
}
