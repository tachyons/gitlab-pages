#!/bin/bash

#
# Downloads, verifies, and extracts the binary dependencies into the right places.
#
# USAGE:
#
#   prepare.sh TAG
#
#     GitLab release tag, e.g. v12.5.0-ubi8
#
# NOTE:
#
#   This script requires `curl`, `gpg`, and `tar`.
#

SCRIPT_HOME="$( cd "${BASH_SOURCE[0]%/*}" > /dev/null 2>&1 && pwd )"

set -euxo pipefail

TAG=${1:-latest}
PACKAGE_NAME="ubi8-build-dependencies-${TAG}.tar"
PACKAGE_HOST='https://gitlab-ubi.s3.us-east-2.amazonaws.com'
PACKAGE_URL="${PACKAGE_HOST}/${PACKAGE_NAME}"
WORKSPACE="${SCRIPT_HOME}/build"
CACHE_LOCATION='/tmp'

mkdir -p "${WORKSPACE}"
mkdir -p "${CACHE_LOCATION}"

# Download and import GitLab's public key
curl --retry 6 -Lf "${PACKAGE_HOST}/gpg" | gpg --import

# Download UBI dependencies package and its signature.
# Cache the package but always download the signature.
curl --retry 6 -Lf "${PACKAGE_URL}.asc" -o "${WORKSPACE}/${PACKAGE_NAME}.asc"
if [ ! -f "${CACHE_LOCATION}/${PACKAGE_NAME}" ]; then
  curl --retry 6 -Lf "${PACKAGE_URL}" -o "${CACHE_LOCATION}/${PACKAGE_NAME}"
fi
cp "${CACHE_LOCATION}/${PACKAGE_NAME}" "${WORKSPACE}/${PACKAGE_NAME}"

# Verify the package integrity
gpg --verify "${WORKSPACE}/${PACKAGE_NAME}.asc" "${WORKSPACE}/${PACKAGE_NAME}"

# Extract UBI dependencies and move them to build contexts
tar -xvf "${WORKSPACE}/${PACKAGE_NAME}" -C "${WORKSPACE}"
rm "${WORKSPACE}/${PACKAGE_NAME}" "${WORKSPACE}/${PACKAGE_NAME}.asc"
for ARCHIVE in $(ls ${WORKSPACE}); do
  TARGET=${ARCHIVE%*.tar.gz}
  cp "${WORKSPACE}/${ARCHIVE}" "${SCRIPT_HOME}/../../${TARGET%*-ee}"
done

# Apply special cases
cp "${WORKSPACE}/gitlab-shell.tar.gz" "${SCRIPT_HOME}/../../gitaly"
cp "${WORKSPACE}/gitlab-python.tar.gz" "${SCRIPT_HOME}/../../gitlab-task-runner"
cp "${WORKSPACE}/gitlab-python.tar.gz" "${SCRIPT_HOME}/../../gitlab-webservice"
