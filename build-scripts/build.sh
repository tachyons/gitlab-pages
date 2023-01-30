#!/bin/bash
# Images that are built nightly on default branch
declare -a nightly_builds=(
  gitlab-rails-ee gitlab-rails-ce
  gitlab-webservice-ce gitlab-webservice-ee
  gitlab-sidekiq-ee gitlab-sidekiq-ce
  gitlab-workhorse-ce gitlab-workhorse-ee
  gitaly gitlab-shell
  gitlab-kas
)

# List of all images that are "final" production images
# Loaded fron CHECKOUT/ci_files/final_images.yml
declare -a final_images=( $(ruby -ryaml -e "puts YAML.safe_load(File.read('ci_files/final_images.yml'))['.final_images'].map {|k| k['job']}.join(' ')") )

function _containsElement () {
  local e match="$1"
  shift
  for e; do [[ "$e" == "$match" ]] && return 0; done
  return 1
}

function is_nightly(){
  [ -n "$NIGHTLY" ] && $(_containsElement $CI_JOB_NAME ${nightly_builds[@]})
}

function is_default_branch(){
  [ "$CI_COMMIT_REF_NAME" == "$CI_DEFAULT_BRANCH" ]
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
  popd

  if needs_build; then
    pushd $(get_trimmed_job_name) # enter image directory

    if [ ! -f "Dockerfile${DOCKERFILE_EXT}" ]; then
      echo "Skipping $(get_trimmed_job_name): Dockerfile${DOCKERFILE_EXT} does not exist."
      popd # be sure to reset working directory
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
      echo "NOTICE: docker cache image enabled, attempting '${CACHE_IMAGE}'"
      if ! $(docker pull $CACHE_IMAGE > /dev/null); then
        if is_stable || is_tag ; then
          echo "NOTICE: docker cache image unavailable, disabled for tags and stable branches"
          CACHE_IMAGE=""
        else
          echo "NOTICE: docker cache image unavailable, attempting to use '${CI_DEFAULT_BRANCH}'"
          CACHE_IMAGE="$CI_REGISTRY_IMAGE/${CI_JOB_NAME#build:*}:${CI_DEFAULT_BRANCH}${IMAGE_TAG_EXT}"
          if ! $(docker pull $CACHE_IMAGE >/dev/null); then
            echo "NOTICE: docker cache image unavailable, disabling"
            CACHE_IMAGE=""
          fi
        fi
      fi

      if [ -n "${CACHE_IMAGE}" ]; then
        echo "NOTICE: docker cache image in use"
        DOCKER_ARGS+=(--cache-from $CACHE_IMAGE)
      fi
    fi

    # Add build image argument for UBI build stage
    if [ "${UBI_BUILD_IMAGE}" = 'true' ]; then
      [ -z "${BUILD_IMAGE}" ] && export BUILD_IMAGE="${CI_REGISTRY_IMAGE}/gitlab-ubi-builder:master-ubi8"
      DOCKER_ARGS+=(--build-arg BUILD_IMAGE="${BUILD_IMAGE}")
    fi

    if [ "${UBI_PIPELINE}" = 'true' ]; then
      DOCKER_ARGS+=(--build-arg DNF_OPTS="${DNF_OPTS:-}")
    fi

    if [ "${FIPS_PIPELINE}" = 'true' ]; then
      DOCKER_ARGS+=(--build-arg FIPS_MODE="${FIPS_MODE}")
    fi

    openshift_labels=()
    openshift_file_name=
    if [ "${FIPS_PIPELINE}" = 'true' ] && [ -f openshift.metadata.fips ]; then
      openshift_file_name=openshift.metadata.fips
    elif [ "${UBI_PIPELINE}" = 'true' ] && [ -f openshift.metadata.ubi8 ]; then
      openshift_file_name=openshift.metadata.ubi8
    else
      openshift_file_name=openshift.metadata
    fi
    if [ -f $openshift_file_name ]; then
      while read -r label; do
        openshift_labels+=(--label "${label}")
      done < $openshift_file_name
    fi

    # Build new image and Push unless it is a UBI build image
    if [ ! "${UBI_BUILD_IMAGE}" = 'true' ]; then
      echo "Not a UBI build image, will build and push the image with computed CONTAINER_VERSION as the image tag."
      BUILD_TYPE='buildx build --push'
    else
      BUILD_TYPE='build'
    fi

    docker $BUILD_TYPE --build-arg CI_REGISTRY_IMAGE=$CI_REGISTRY_IMAGE -t "$CI_REGISTRY_IMAGE/${CI_JOB_NAME#build:*}:$CONTAINER_VERSION${IMAGE_TAG_EXT}" "${DOCKER_ARGS[@]}" -f Dockerfile${DOCKERFILE_EXT} ${DOCKER_BUILD_CONTEXT:-.} "${openshift_labels[@]}"

    # Output "Final Image Size: %d" (gitlab-org/charts/gitlab#1267)
    docker inspect "$CI_REGISTRY_IMAGE/${CI_JOB_NAME#build:*}:$CONTAINER_VERSION${IMAGE_TAG_EXT}" \
      | awk '/"Size": ([0-9]+)[,]?/{ printf "Final Image Size: %d\n", $2 }'

    popd # exit image directory

  fi

  # Record image repository and tag unless it is a UBI build image
  if [ ! "${UBI_BUILD_IMAGE}" = 'true' ]; then
    echo "${CI_JOB_NAME#build:*}:$CONTAINER_VERSION${IMAGE_TAG_EXT}" > "artifacts/images/${CI_JOB_NAME#build:*}.txt"
  fi
}

function tag_and_push(){
  local edition=$1
  local mirror_image_name=$2
  local source_image="${CI_REGISTRY_IMAGE}/${CI_JOB_NAME#build:*}:${CONTAINER_VERSION}${IMAGE_TAG_EXT}"
  local target_image="${CI_REGISTRY_IMAGE}/${CI_JOB_NAME#build:*}:${edition}"

  # If mirror image name is defined, then override the target image name.
  if [ -n "${mirror_image_name}" ]; then
    target_image="${CI_REGISTRY_IMAGE}/${mirror_image_name#build:*}:$edition"
  fi

  # Tag and push unless it is a UBI build image
  if [ ! "${UBI_BUILD_IMAGE}" = 'true' -a -f "$(get_trimmed_job_name)/Dockerfile${DOCKERFILE_EXT}" ]; then
    docker tag "${source_image}" "${target_image}"
    docker push "${target_image}"
  fi
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

function is_auto_deploy(){
  [[ $CI_COMMIT_BRANCH =~ $AUTO_DEPLOY_BRANCH_REGEX ]] || [[ $CI_COMMIT_TAG =~ $AUTO_DEPLOY_TAG_REGEX ]]
}

function is_regular_tag(){
  is_tag && ! is_auto_deploy
}

# When `push_tags` is called with `${GITLAB_REF_SLUG}${IMAGE_TAG_EXT}` as
# arguments, we will have something like `v15.1.3-ee-ubi8` or
# `v15.1.3-ee-fips`. We need to strip off the `-ee` part from it.
function trim_edition(){
  echo $1 | sed -e "s/-.e\(-ubi8\|-fips\)\?$/\1/"
}

function trim_tag(){
  echo $(trim_edition $1) | sed -e "s/^v//"
}

function is_final_image(){
  [[ ${final_images[*]} =~ ${CI_JOB_NAME#build:*} ]]
}

function push_tags(){
  if [ ! -f "$(get_trimmed_job_name)/Dockerfile${DOCKERFILE_EXT}" ]; then
    echo "Skipping $(get_trimmed_job_name): Dockerfile${DOCKERFILE_EXT} does not exist."
    return 0
  fi

  local mirror_image_name=$2

  # If a version has been specified and we are on master branch or a
  # non-auto-deploy tag, we use the specified version.
  if [ -n "$1" ] && (is_default_branch || is_regular_tag); then
    echo "Pipeline running against default branch or stable tag. Using specified version as the image tag."
    local edition=$1

    # If on a non-auto-deploy tag pipeline, we can trim the `-ee` suffixes.
    if is_regular_tag; then
      edition=$(trim_edition $edition)
    fi

    version_to_tag=$edition
  elif is_regular_tag; then
    echo "Pipeline running against stable tag and no version specified. Using git tag to as the image tag."
    # If no version is specified, but on a non-auto-deploy tag pipeline, we use
    # the trimmed tag.
    trimmed_tag=$(trim_edition $CI_COMMIT_TAG)

    version_to_tag=$trimmed_tag
  elif [ -z "$1" ]; then
    echo "No version specified. Using commit ref slug as the image tag."
    # If no version was specified at all, we use the slug.
    version_to_tag=${CI_COMMIT_REF_SLUG}${IMAGE_TAG_EXT}
  else
    # If a version was specified on any other scenarios - branch builds or
    # auto-deploy tag builds, we ignore it as we don't want to overwrite an
    # existing versioned image. Since we always call `push_tags` without a
    # version before calling it with a version, the image would've been already
    # tagged with commit ref slug so we need not try to do it again.
    echo "Pipeline running against feature branch or auto-deploy tag. Not tagging the image with specified version."
    version_to_tag=""
  fi

  if [ -n "$version_to_tag" ]; then
    tag_and_push $version_to_tag $mirror_image_name

    # Append the newly pushed tags also to the artifact list
    echo "${CI_JOB_NAME#build:*}:${version_to_tag}" >> "artifacts/images/${CI_JOB_NAME#build:*}.txt"

    # if this is a final image, record it separately.
    if is_final_image; then
      echo "${CI_JOB_NAME#build:*}:${version_to_tag}" > "artifacts/final/${CI_JOB_NAME#build:*}.txt"
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
    echo "==== Assets Summary ===="
    du -hd2 "${ASSETS_DIR}/assets"
    tar -czf "${ASSETS_DIR}.tar.gz" -C "${ASSETS_DIR}/assets" .
    echo $(sha256sum "${ASSETS_DIR}.tar.gz") $(du -h "${ASSETS_DIR}.tar.gz" | awk '{print $1}')
    rm -rf "${ASSETS_DIR}"
    echo "==== Cleanup UBI artifacts"
    du -hd1 --all artifacts/ubi/*.tar.gz
    for tarball in artifacts/ubi/*.tar.gz ; do
      if [ "${tarball}" != "${ASSETS_DIR}.tar.gz" ]; then
        rm -f "${tarball}"
      fi
    done
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
    mock_tags_from_assets
  fi
}

# mock_tags_from_assets
# To support UBI having assets versus artifact containers, we checksum
# the assets tarballs, and use these as the "container_version" content.
mock_tags_from_assets() {
  if [ "${UBI_PIPELINE}" = 'true' ]; then
    trimmed_job_name=$(get_trimmed_job_name)
    assets="${trimmed_job_name}/*.tar.gz"
    shopt -s nullglob
    for asset in $assets; do
      container=$(basename $asset)
      false_tag="artifacts/container_versions/${container%.tar.gz}_tag.txt"
      sha256sum $asset > "${false_tag}"
    done
    shopt -u nullglob
  fi
}

## record_stable_image
# pull a base image at a tag, record the tag's digest into container_versions
record_stable_image() {
  image=$1
  name=$(image_root_name ${image})
  docker pull ${image}
  # Emulate `skopeo inspect docker://${FULL_IMAGE} | jq -r '.Digest'`
  image_digest=$(docker inspect --format '{{ join .RepoDigests " , " }}' ${image} | cut -d'@' -f2)
  echo -n "${image_digest}" > "artifacts/container_versions/${name}.txt"
}

## image_root_name
# return the "basename" of an image
# - docker.io/library/alpine:3.15 => alpine
# - docker.io/library/debian:bullseye-slim => debian
image_root_name() {
  IMAGE=$1
  IMAGE=${IMAGE##*/} # remove all leading slashes
  IMAGE=${IMAGE%%:*} # remove longest from end, with :
  IMAGE=${IMAGE%%@*} # remove longest from end, with @
  echo -n $IMAGE
}

## populate_stable_image_vars
# export the various environment variables surrounding stable-ized distribtion images
# If distributions have entries in `container_verions`, export those for use by CI
# and/or scripting
populate_stable_image_vars() {
  # update DEBIAN_IMAGE to full origin & digest
  if [ -f artifacts/container_versions/debian.txt ]; then
    export DEBIAN_DIGEST=$(cat artifacts/container_versions/debian.txt) ;
    export DEBIAN_IMAGE="${DEPENDENCY_PROXY}${DEBIAN_IMAGE}@${DEBIAN_DIGEST}" ;
    export DEBIAN_BUILD_ARGS="--build-arg DEBIAN_IMAGE=${DEBIAN_IMAGE}"
    echo "DEBIAN_BUILD_ARGS: ${DEBIAN_BUILD_ARGS}"
  fi
  # update DEBIAN_IMAGE to full origin & digest
  if [ -f artifacts/container_versions/ubi.txt ]; then
    export UBI_DIGEST=$(cat artifacts/container_versions/ubi.txt) ;
    export UBI_IMAGE="${UBI_IMAGE}@${UBI_DIGEST}" ;
    export UBI_BUILD_ARGS="--build-arg UBI_IMAGE=${UBI_IMAGE}"
    echo "UBI_BUILD_ARGS: ${UBI_BUILD_ARGS}"
  fi
  # update ALPINE_IMAGE to full origin & digest
  if [ -f artifacts/container_versions/alpine.txt ]; then
    export ALPINE_DIGEST=$(cat artifacts/container_versions/alpine.txt) ;
    export ALPINE_IMAGE="${DEPENDENCY_PROXY}${ALPINE_IMAGE}@${ALPINE_DIGEST}" ;
    export ALPINE_BUILD_ARGS="--build-arg ALPINE_IMAGE=${ALPINE_IMAGE}"
    echo "ALPINE_BUILD_ARGS: ${ALPINE_BUILD_ARGS}"
  fi
}

## list_artifacts
# helper function to list any/all contents of incoming/outgoing artifacts
# input: subdirectory to `artifacts` on which to focus
list_artifacts() {
    subdirectory=$1
    directory="artifacts"
    if [ -d "${directory}/${subdirectory}" ]; then 
      directory="${directory}/${subdirectory}"
    fi
    echo "==== Artifacts Summary ===="
    du -hd2 --all "${directory}"
    echo "==========================="
}

