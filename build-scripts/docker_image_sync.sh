#!/bin/sh

if [ $# -eq 0 ] ; then
  echo 'No file specified'
  exit 0
fi

set -e

component_file=$1

COM_REGISTRY=${COM_REGISTRY:-"registry.gitlab.com"}
COM_CNG_PROJECT=${COM_CNG_PROJECT:-"gitlab-org/build/cng"}

echo "${CI_JOB_TOKEN}" | docker login -u "gitlab-ci-token" --password-stdin "${CI_REGISTRY}"
echo "Pulling images from dev registry"
while IFS=: read -r component tag; do
  docker pull -q "${CI_REGISTRY_IMAGE}/${component}:${tag}"
  docker tag "${CI_REGISTRY_IMAGE}/${component}:${tag}" "${COM_REGISTRY}/${COM_CNG_PROJECT}/${component}:${tag}"
done < "${component_file}"

echo "${COM_REGISTRY_PASSWORD}" | docker login -u "${CI_REGISTRY_USER}" --password-stdin "${COM_REGISTRY}"
echo "Pushing images to com registry"

while IFS=: read -r component tag; do
  docker push "${COM_REGISTRY}/${COM_CNG_PROJECT}/${component}:${tag}"
  echo "${COM_REGISTRY}/${COM_CNG_PROJECT}/${component}:${tag}" >> artifacts/cng_images.txt
done < "${component_file}"
