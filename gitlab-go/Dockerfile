ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG TAG="master"
ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/gitlab-ruby"

FROM ${FROM_IMAGE}:${TAG}

ARG GO_VERSION=1.17.6
ARG BUILD_DIR=/tmp/build

ENV GO_URL="https://storage.googleapis.com/golang/go${GO_VERSION}.linux-amd64.tar.gz"

# install go
RUN mkdir ${BUILD_DIR} \
  && cd ${BUILD_DIR} \
  && curl --retry 6 -sfo golang.tar.xz ${GO_URL} \
  && tar -xf golang.tar.xz \
  && mv ${BUILD_DIR}/go /usr/local/go \
  && rm golang.tar.xz \
  && rm -rf /var/lib/apt/lists/* \
  && ln -sf /usr/local/go/bin/go /usr/local/go/bin/gofmt /usr/local/go/bin/godoc /usr/local/bin/ \
  && rm -rf ${BUILD_DIR}
