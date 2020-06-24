#!/bin/sh

set -e

cd artifacts/

ASSETS_DIR="ubi8-build-dependencies-${UBI_RELEASE_TAG}"
ASSETS_PACK="${ASSETS_DIR}.tar"
ASSETS_URL="http://${UBI_ASSETS_AWS_BUCKET}.s3.amazonaws.com/${ASSETS_PACK}"
gpg --batch --quiet --yes --armor --export --output gpg
tar -cvf ${ASSETS_PACK} -C ubi .
gpg --passphrase "${GPG_KEY_PASSPHRASE}" --batch --quiet --yes --armor --pinentry-mode loopback --detach-sign ${ASSETS_PACK}
sha256sum ${ASSETS_PACK} | awk '{print $1}' > "${ASSETS_PACK}.sha256"
aws s3 cp --quiet --acl public-read --content-type application/x-pem-file gpg "s3://${UBI_ASSETS_AWS_BUCKET}/gpg"
aws s3 cp --quiet --acl public-read --content-type application/x-pem-file ${ASSETS_PACK}.asc "s3://${UBI_ASSETS_AWS_BUCKET}/${ASSETS_PACK}.asc"
aws s3 cp --quiet --acl public-read --content-type application/x-pem-file ${ASSETS_PACK}.sha256 "s3://${UBI_ASSETS_AWS_BUCKET}/${ASSETS_PACK}.sha256"
aws s3 cp --quiet --acl public-read --content-type application/x-tar ${ASSETS_PACK} "s3://${UBI_ASSETS_AWS_BUCKET}/${ASSETS_PACK}"
curl --retry 6 -f -H "PRIVATE-TOKEN:${UBI_RELEASE_PAT}" -H 'Content-Type:application/json' --data \
    "$(printf \
      '{"tag_name":"%s","ref":"%s","name":"%s","description":"%s","assets":{"links":[{"name":"%s","url":"%s"},{"name":"%s","url":"%s"},{"name":"%s","url":"%s"}]}}' \
      "${UBI_RELEASE_TAG}" \
      "${UBI_RELEASE_TAG}" \
      "Release ${UBI_RELEASE_TAG}" \
      "Binary dependencies for building UBI-based images for Cloud-Native GitLab." \
      "${ASSETS_PACK}" \
      "${ASSETS_URL}" \
      "${ASSETS_PACK}.asc" \
      "${ASSETS_URL}.asc" \
      "${ASSETS_PACK}.sha256" \
      "${ASSETS_URL}.sha256" \
    )" \
    "${RELEASE_API}"

# Exclude duplicates from the cache
rm -r ubi
