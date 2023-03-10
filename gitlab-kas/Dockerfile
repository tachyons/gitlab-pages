ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG TAG="master"
ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/gitlab-go"

FROM cgr.dev/chainguard/go:1.19-dev as builder

ARG GITLAB_KAS_VERSION=v14.5.0
ARG GITLAB_NAMESPACE=gitlab-org
ARG BUILD_DIR=/tmp/build
ARG FETCH_ARTIFACTS_PAT
ARG CI_API_V4_URL="https://gitlab.com/api/v4"

# install kas
RUN apk add curl \
    && echo "Downloading source code from ${CI_API_V4_URL}/projects/${GITLAB_NAMESPACE}%2Fcluster-integration%2Fgitlab-agent/repository/archive.tar.bz2?sha=${GITLAB_KAS_VERSION}" \
    && curl -fL --retry 6 --header "PRIVATE-TOKEN: ${FETCH_ARTIFACTS_PAT}" -o gitlab-kas.tar.bz2 "${CI_API_V4_URL}/projects/${GITLAB_NAMESPACE}%2Fcluster-integration%2Fgitlab-agent/repository/archive.tar.bz2?sha=${GITLAB_KAS_VERSION}" \
    && mkdir -p "${BUILD_DIR}" \
    && tar -xjf gitlab-kas.tar.bz2 -C "${BUILD_DIR}" --strip-components=1 \
    && rm gitlab-kas.tar.bz2 \
    && cd "${BUILD_DIR}" \
    && echo "Building kas binary" \
    && GIT_COMMIT="${GITLAB_KAS_VERSION}" GIT_TAG="${GITLAB_KAS_VERSION}" TARGET_DIRECTORY=/usr/bin make kas

## FINAL IMAGE ##

FROM cgr.dev/chainguard/wolfi-base:latest

COPY --from=builder /usr/bin/kas /usr/bin/kas

ENTRYPOINT ["/usr/bin/kas"]
