#!/bin/sh
# install-cfssl.sh
set -e

# Version & SHA2 of the checksums list
CFSSL_VERSION=${CFSSL_VERSION:-1.6.1}
CFSSL_CHECKSUM_SHA256="89e600cd5203a025f8b47c6cd5abb9a74b06e3c7f7f7dd3f5b2a00975b15a491"
# Download and install CFSSL from https://github.com/cloudflare/cfssl/releases
CFSSL_PKG_URL="https://github.com/cloudflare/cfssl/releases/download/v${CFSSL_VERSION}"
CFSSL_LICENSE="https://raw.githubusercontent.com/cloudflare/cfssl/v${CFSSL_VERSION}/LICENSE"
# we want linux_amd64 by default
CFSSL_PLATFORM=${CFSSL_PLATFORM:-linux_amd64}
# /usr/local/bin is in PATH for Alpine & RedHat UBI
CFSSL_BIN=${CFSSL_BIN:-/usr/local/bin}
CFSSL_ITEMS="cfssl cfssljson"

CWD=`pwd`
WORK_DIR=$(mktemp -d)
cd $WORK_DIR

# Fetch CHECKSUMS
echo "Fecthing CHECKSUMS for ${CFSSL_VERSION}"
CHECKSUM_FILE="cfssl_${CFSSL_VERSION}_checksums.txt"
curl --retry 6 -fJLO "${CFSSL_PKG_URL}/${CHECKSUM_FILE}"

echo "Fetching items: ${CFSSL_ITEMS}"
for item in ${CFSSL_ITEMS} ; do
  ITEM_PATH="${item}_${CFSSL_VERSION}_${CFSSL_PLATFORM}"
  ITEM_URL="${CFSSL_PKG_URL}/${ITEM_PATH}"
  echo "Fetching '${item}' from '${ITEM_URL}'"
  curl --retry 6 -fJLO  "$ITEM_URL"
  grep ${ITEM_PATH} ${CHECKSUM_FILE} >> checksums.txt
done

echo "Verifying checksums"
sha256sum -c checksums.txt

for item in ${CFSSL_ITEMS} ; do
  echo "Placing '${item}' in '${CFSSL_BIN}'"
  mv "${item}_${CFSSL_VERSION}_${CFSSL_PLATFORM}" ${CFSSL_BIN}/${item}
  chmod +x ${CFSSL_BIN}/${item}
done

echo "Fetching LICENSE"
curl -fJLo ${CFSSL_BIN}/cfssl.LICENSE "${CFSSL_LICENSE}"

cd $CWD
rm -rf $WORK_DIR
