ARG DEBIAN_IMAGE=debian:bullseye-slim

FROM ${DEBIAN_IMAGE}

ARG GO_VERSION=1.18.7
ARG BUILD_DIR=/tmp/build

ENV GO_URL="https://storage.googleapis.com/golang/go${GO_VERSION}.linux-amd64.tar.gz"

# Install go
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        curl \
        ca-certificates \
        bzip2 \
    && ln -s /etc/ssl/certs/ca-certificates.crt /usr/lib/ssl/cert.pem \
    && mkdir ${BUILD_DIR} \
    && cd ${BUILD_DIR} \
    && curl --retry 6 -sfo golang.tar.xz ${GO_URL} \
    && tar -xf golang.tar.xz \
    &&  mv ${BUILD_DIR}/go /usr/local/go \
    &&  rm golang.tar.xz \
    &&  rm -rf /var/lib/apt/lists/* \
    &&  ln -sf /usr/local/go/bin/go /usr/local/go/bin/gofmt /usr/local/go/bin/godoc /usr/local/bin/ \
    &&  rm -rf ${BUILD_DIR}
