#!/bin/sh

set -e

usage() {
  cat <<EOF
Usage: $0 <image file> <gitlab-com|artifact-registry>

Examples:

  # Syncs images to registry.gitlab.com
  # registry.gitlab.com/gitlab-org/build/cng

  $0 image_versions.txt gitlab-com

  # Sync images to an artifact registry on GCP
  # us-east1-docker.pkg.dev/gitlab-com-artifact-registry/images

  $0 image_versions.txt artifact-registry
EOF
}

if [ $# -ne 2 ] ; then
  usage
  exit 1
fi

component_file=$1
registry_destination=$2

case "$registry_destination" in
  gitlab-com)
    if [ -z $COM_REGISTRY_PASSWORD ]; then
      echo 'Skipping sync because $COM_REGISTRY_PASSWORD is not set in environment' 1>&2
      exit 0
    fi

    DEST_REGISTRY=${COM_REGISTRY:-"registry.gitlab.com"}
    DEST_PATH=${COM_CNG_REGISTRY_PATH:-"gitlab-org/build/cng"}
    echo "${COM_REGISTRY_PASSWORD}" | docker login -u "${CI_REGISTRY_USER}" --password-stdin "${DEST_REGISTRY}"
    ;;
  artifact-registry)
    if [ -z $ARTIFACT_REGISTRY_SA_FILE ]; then
      echo 'Skipping sync because $ARTIFACT_REGISTRY_SA_FILE is not set in environment' 1>&2
      exit 0
    fi

    DEST_REGISTRY=${ARTIFACT_REGISTRY:-"us-east1-docker.pkg.dev"}
    DEST_PATH=${ARTIFACT_REGISTRY_PATH:-"gitlab-com-artifact-registry/images"}
    cat "${ARTIFACT_REGISTRY_SA_FILE}" | docker login -u _json_key  --password-stdin "${DEST_REGISTRY}"
    ;;
  *)
    echo "Invalid destination: $registry_destination" 1>&2
    usage
    exit 1
  ;;
esac

echo "${CI_JOB_TOKEN}" | docker login -u "gitlab-ci-token" --password-stdin "${CI_REGISTRY}"

while IFS=: read -r component tag; do
  src="${CI_REGISTRY_IMAGE}/${component}:${tag}"
  dest="${DEST_REGISTRY}/${DEST_PATH}/${component}:${tag}"
  echo "Copying $src to $dest"
  skopeo copy --multi-arch=all "docker://$src" "docker://$dest"
  echo "$dest" >> artifacts/${registry_destination}-images.txt
done < "${component_file}"
