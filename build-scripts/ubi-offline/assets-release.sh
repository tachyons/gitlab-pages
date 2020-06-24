#!/bin/sh

set -e

cd artifacts/

ASSETS_DIR="ubi8-build-dependencies-${UBI_RELEASE_TAG}"
ASSETS_PACK="${ASSETS_DIR}.tar"
ASSETS_URL_PREFIX="http://${UBI_ASSETS_AWS_BUCKET}.s3.amazonaws.com"
gpg --batch --quiet --yes --armor --export --output gpg

links=()

addLink() {
  local urlPrefix="${1}"
  local asset="${2}"
  assets+=("$(printf '{"name":"%s","url":"%s"}' "${urlPrefix}/${asset}" "${asset}")")
}

tar -cvf ${ASSETS_PACK} -C ubi .
gpg --passphrase "${GPG_KEY_PASSPHRASE}" --batch --quiet --yes --armor --pinentry-mode loopback --detach-sign ${ASSETS_PACK}
sha256sum ${ASSETS_PACK} | awk '{print $1}' > "${ASSETS_PACK}.sha256"
aws s3 cp --quiet --acl public-read --content-type application/x-pem-file gpg "s3://${UBI_ASSETS_AWS_BUCKET}/gpg"
aws s3 cp --quiet --acl public-read --content-type application/x-pem-file ${ASSETS_PACK}.asc "s3://${UBI_ASSETS_AWS_BUCKET}/${ASSETS_PACK}.asc"
aws s3 cp --quiet --acl public-read ${ASSETS_PACK}.sha256 "s3://${UBI_ASSETS_AWS_BUCKET}/${ASSETS_PACK}.sha256"
aws s3 cp --quiet --acl public-read --content-type application/x-tar ${ASSETS_PACK} "s3://${UBI_ASSETS_AWS_BUCKET}/${ASSETS_PACK}"
addLink $ASSETS_URL_PREFIX $ASSETS_PACK
addLink $ASSETS_URL_PREFIX "${ASSETS_PACK}.asc"
addLink $ASSETS_URL_PREFIX "${ASSETS_PACK}.sha256"

for asset in ubi/*.tar.gz
do
  gpg --passphrase "${GPG_KEY_PASSPHRASE}" --batch --quiet --yes --armor --pinentry-mode loopback --detach-sign ${asset}
  sha256sum ${asset} | awk '{print $1}' > "${asset}.sha256"
  aws s3 cp --quiet --acl public-read --content-type application/x-pem-file ${asset}.asc "s3://${UBI_ASSETS_AWS_BUCKET}/${ASSETS_DIR/}${asset}.asc"
  aws s3 cp --quiet --acl public-read ${asset}.sha256 "s3://${UBI_ASSETS_AWS_BUCKET}/${ASSETS_DIR/}${asset}.sha256"
  aws s3 cp --quiet --acl public-read --content-type application/gzip ${asset} "s3://${UBI_ASSETS_AWS_BUCKET}/${ASSETS_DIR}/${asset}"
  addLink "${ASSETS_URL_PREFIX}/${ASSETS_DIR}" $asset
  addLink "${ASSETS_URL_PREFIX}/${ASSETS_DIR}" "${asset}.asc"
  addLink "${ASSETS_URL_PREFIX}/${ASSETS_DIR}"  "${asset}.sha256"
done

curl --retry 6 -f -H "PRIVATE-TOKEN:${UBI_RELEASE_PAT}" -H 'Content-Type:application/json' --data \
    "$(printf \
      '{"tag_name":"%s","ref":"%s","name":"%s","description":"%s","assets":{"links":[%s]}}' \
      "${UBI_RELEASE_TAG}" \
      "${UBI_RELEASE_TAG}" \
      "Release ${UBI_RELEASE_TAG}" \
      "Binary dependencies for building UBI-based images for Cloud-Native GitLab." \
      "$(printf '$s,' "${links[@]}")" \
    )" \
    "${RELEASE_API}"

# Exclude duplicates from the cache
rm -r ubi
