#!/usr/bin/env bash

set -euo pipefail

if [ -z "${UBI_RELEASE_TAG:-}" ] || \
   [ -z "${UBI_ASSETS_AWS_BUCKET:-}" ] || \
   [ -z "${RELEASE_API:-}" ] || \
   [ -z "${UBI_RELEASE_PAT:-}" ] || \
   [ -z "${GPG_KEY_PASSPHRASE:-}" ]; then
  echo "UBI_RELEASE_TAG, UBI_ASSETS_AWS_BUCKET, RELEASE_API, UBI_RELEASE_PAT and GPG_KEY_PASSPHRASE are required"
  exit 0
fi

cd artifacts/

ASSETS_DIR="ubi8-build-dependencies-${UBI_RELEASE_TAG}"
ASSETS_PACK="${ASSETS_DIR}.tar"
ASSETS_URL_PREFIX="http://${UBI_ASSETS_AWS_BUCKET}.s3.amazonaws.com"

links=()

joinBy() {
  local IFS="${1}"
  shift
  echo "$*"
}

addLink() {
  local urlPrefix="${1}"
  shift
  for asset in "$@"
  do
    links+=("$(printf '{"name":"%s","url":"%s"}' "${asset}" "${urlPrefix}/${asset}" )")
  done
}

signAsset() {
  gpg --passphrase "${GPG_KEY_PASSPHRASE}" --batch --quiet --yes --armor --pinentry-mode loopback --detach-sign "${1}"
}

sumAsset() {
  local asset="${1}"
  sha256sum "${asset}" | awk '{print $1}' > "${asset}.sha256"
}

s3Copy() {
  local asset="${1}"
  local path="${2}"
  local type="${3:-text/plain}"
  aws s3 cp --quiet --acl public-read --content-type "${type}" "${asset}" "s3://${UBI_ASSETS_AWS_BUCKET}/${path}"
}

releaseAsset() {
  local asset="${1}"
  local path="${2}"
  local type="${3:-text/plain}"
  signAsset "${asset}"
  sumAsset "${asset}"
  s3Copy "${asset}.asc" "${path}.asc" "application/x-pem-file"
  s3Copy "${asset}.sha256" "${path}.sha256"
  s3Copy "${asset}" "${path}" "${type}"
}

gpg --batch --quiet --yes --armor --export --output gpg
tar -cvf "${ASSETS_PACK}" -C ubi .
s3Copy 'gpg' 'gpg' 'application/x-pem-file'
releaseAsset "${ASSETS_PACK}" "${ASSETS_PACK}" "application/x-tar"
addLink "${ASSETS_URL_PREFIX}" "${ASSETS_PACK}" "${ASSETS_PACK}.asc" "${ASSETS_PACK}.sha256"

for asset in ubi/*.tar.gz
do
  asset_name="${asset##*/}"
  releaseAsset "${asset}" "${ASSETS_DIR}/${asset_name}" "application/gzip"
  addLink "${ASSETS_URL_PREFIX}/${ASSETS_DIR}" "${asset_name}" "${asset_name}.asc" "${asset_name}.sha256"
done

curl --retry 6 -f -H "PRIVATE-TOKEN:${UBI_RELEASE_PAT}" -H 'Content-Type:application/json' --data \
    "$(printf \
      '{"tag_name":"%s","ref":"%s","name":"%s","description":"%s","assets":{"links":[%s]}}' \
      "${UBI_RELEASE_TAG}" \
      "${UBI_RELEASE_TAG}" \
      "Release ${UBI_RELEASE_TAG}" \
      "Binary dependencies for building UBI-based images for Cloud-Native GitLab." \
      "$(joinBy ',' "${links[@]}")" \
    )" \
    "${RELEASE_API}"

# Exclude duplicates from the cache
rm -r ubi
