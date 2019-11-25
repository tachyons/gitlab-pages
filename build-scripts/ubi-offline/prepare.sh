#!/bin/bash

# NOTICE: This script requires `curl`, `gpg`, and `tar`.

set -euxo pipefail

TAG=${1:-latest}
PACKAGE_NAME="ubi8-build-dependencies-${TAG}.tar"
PACKAGE_HOST="https://gitlab-ubi.s3.us-east-2.amazonaws.com"
PACKAGE_URL="${PACKAGE_HOST}/${PACKAGE_NAME}"
WORKSPACE="prepare"

mkdir ${WORKSPACE}
trap "rm -rf ${WORKSPACE}" EXIT

# Download and import GitLab's public key
curl -Lf "${PACKAGE_HOST}/gpg" | gpg --import

# Download UBI dependencies package and it signature
curl -Lf "${PACKAGE_URL}.asc" -o "${WORKSPACE}/${PACKAGE_NAME}.asc"
curl -Lf "${PACKAGE_URL}" -o "${WORKSPACE}/${PACKAGE_NAME}"

# Verify the package integrity
gpg --verify "${WORKSPACE}/${PACKAGE_NAME}.asc" "${WORKSPACE}/${PACKAGE_NAME}"

# Extract UBI dependencies and move them to build contexts
tar -xvf "${WORKSPACE}/${PACKAGE_NAME}" -C "${WORKSPACE}"
rm "${WORKSPACE}/${PACKAGE_NAME}" "${WORKSPACE}/${PACKAGE_NAME}.asc"
for ARCHIVE in $(ls ${WORKSPACE}); do
  TARGET=${ARCHIVE%*.tar.gz}
  cp "${WORKSPACE}/${ARCHIVE}" "${TARGET%*-ee}"
done

# Apply special cases
cp "${WORKSPACE}/gitlab-shell.tar.gz" gitaly
cp "${WORKSPACE}/gitlab-python.tar.gz" gitlab-task-runner
cp "${WORKSPACE}/gitlab-python.tar.gz" gitlab-unicorn
cp "${WORKSPACE}/kubectl.tar.gz" gitlab-redis-ha
