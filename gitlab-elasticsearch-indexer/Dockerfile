ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG TAG="master"
ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/gitlab-go"

FROM ${FROM_IMAGE}:${TAG}

ARG BUILD_DIR=/tmp/build
ARG GITLAB_ELASTICSEARCH_INDEXER_VERSION="v1.1.0"
ARG GITLAB_NAMESPACE="gitlab-org"
ARG FETCH_ARTIFACTS_PAT
ARG CI_API_V4_URL

RUN buildDeps=' \
  gcc \
  git \
  libicu-dev \
  make \
  pkg-config' \
  && apt-get update \
  && apt-get install -y --no-install-recommends $buildDeps \
  && rm -rf /var/lib/apt/lists/* \
  && mkdir -p ${BUILD_DIR}

RUN cd ${BUILD_DIR} && \
  echo "Downloading source code from ${CI_API_V4_URL}/projects/${GITLAB_NAMESPACE}%2Fgitlab-elasticsearch-indexer/repository/archive.tar.bz2?sha=${GITLAB_ELASTICSEARCH_INDEXER_VERSION}" && \
  curl -f --retry 6 --header "PRIVATE-TOKEN: ${FETCH_ARTIFACTS_PAT}" -o gitlab-elasticsearch-indexer.tar.bz2 "${CI_API_V4_URL}/projects/${GITLAB_NAMESPACE}%2Fgitlab-elasticsearch-indexer/repository/archive.tar.bz2?sha=${GITLAB_ELASTICSEARCH_INDEXER_VERSION}" && \
  tar -xjf gitlab-elasticsearch-indexer.tar.bz2 --strip-components=1 && \
  rm gitlab-elasticsearch-indexer.tar.bz2 && \
  make && \
  make install && \
  cd / && \
  rm -rf "${BUILD_DIR}"
