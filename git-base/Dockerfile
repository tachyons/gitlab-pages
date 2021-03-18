ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG TAG="master"
ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/gitlab-go"

FROM ${FROM_IMAGE}:${TAG}

ARG BUILD_DIR=/tmp/build
ARG GITALY_SERVER_VERSION=v1.83.0
ARG GITLAB_NAMESPACE="gitlab-org"
ARG FETCH_ARTIFACTS_PAT
ARG CI_API_V4_URL

RUN buildDeps=' \
    make \
    cmake \
    gcc \
    g++ \
    git \
    patch \
    libicu-dev \
    libpcre2-dev \
    libcurl4-gnutls-dev \
    pkg-config \
    sudo' \
    && apt-get update \
    && apt-get install -y --no-install-recommends $buildDeps \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p ${BUILD_DIR}

RUN cd ${BUILD_DIR} \
    echo "Downloading source code from ${CI_API_V4_URL}/projects/${GITLAB_NAMESPACE}%2Fgitaly/repository/archive.tar.bz2?sha=${GITALY_SERVER_VERSION}" \
    && curl --retry 6 --header "PRIVATE-TOKEN: ${FETCH_ARTIFACTS_PAT}" -o gitaly.tar.bz2 "${CI_API_V4_URL}/projects/${GITLAB_NAMESPACE}%2Fgitaly/repository/archive.tar.bz2?sha=${GITALY_SERVER_VERSION}" \
    && mkdir gitaly \
    && tar --strip-components 1 -C gitaly -xjf gitaly.tar.bz2 \
    && rm gitaly.tar.bz2 \
    && make -C gitaly git GIT_PREFIX=/usr/local \
    && rm -rf gitaly
