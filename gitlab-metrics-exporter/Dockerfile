ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG TAG="master"
ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/gitlab-go"

FROM ${FROM_IMAGE}:${TAG}

ARG BUILD_DIR=/tmp/build
ARG VERSION
ARG GITLAB_NAMESPACE="gitlab-org"
ARG FETCH_ARTIFACTS_PAT
ARG CI_API_V4_URL
ARG REPO_NAME="gitlab-metrics-exporter"
ARG ARCHIVE_FILE="${REPO_NAME}.tar.bz2"
ARG ARCHIVE_URL="${CI_API_V4_URL}/projects/${GITLAB_NAMESPACE}%2F${REPO_NAME}/repository/archive.tar.bz2?sha=${VERSION}"

RUN apt-get update \
  && apt-get install -y --no-install-recommends make \
  && rm -rf /var/lib/apt/lists/* \
  && mkdir -p ${BUILD_DIR}

RUN cd ${BUILD_DIR} && \
  echo "Downloading source code from ${ARCHIVE_URL}" && \
  curl -f --retry 6 --header "PRIVATE-TOKEN: ${FETCH_ARTIFACTS_PAT}" -o "${ARCHIVE_FILE}" "${ARCHIVE_URL}" && \
  tar -xjf "${ARCHIVE_FILE}" --strip-components=1 && \
  rm "${ARCHIVE_FILE}" && \
  make && \
  make install && \
  cd / && \
  rm -rf "${BUILD_DIR}"
