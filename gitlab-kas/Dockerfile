ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG TAG="master"
ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/gitlab-go"

FROM ${FROM_IMAGE}:${TAG} as builder

ARG GITLAB_KAS_VERSION=v14.5.0
ARG GITLAB_NAMESPACE=gitlab-org
ARG BUILD_DIR=/tmp/build
ARG FETCH_ARTIFACTS_PAT
ARG CI_API_V4_URL

# install debian packages
RUN apt-get update && apt-get install -y build-essential git

# install kas
RUN echo "Downloading source code from ${CI_API_V4_URL}/projects/${GITLAB_NAMESPACE}%2Fcluster-integration%2Fgitlab-agent/repository/archive.tar.bz2?sha=${GITLAB_KAS_VERSION}" \
    && curl -fL --retry 6 --header "PRIVATE-TOKEN: ${FETCH_ARTIFACTS_PAT}" -o gitlab-kas.tar.bz2 "${CI_API_V4_URL}/projects/${GITLAB_NAMESPACE}%2Fcluster-integration%2Fgitlab-agent/repository/archive.tar.bz2?sha=${GITLAB_KAS_VERSION}" \
    && mkdir -p "${BUILD_DIR}" \
    && tar -xjf gitlab-kas.tar.bz2 -C "${BUILD_DIR}" --strip-components=1 \
    && rm gitlab-kas.tar.bz2 \
    && cd "${BUILD_DIR}" \
    && echo "Building kas binary" \
    && GIT_COMMIT="${GITLAB_KAS_VERSION}" GIT_TAG="${GITLAB_KAS_VERSION}" TARGET_DIRECTORY=/usr/bin make kas

# build image is derived from debian 11, so we use that for run image
FROM gcr.io/distroless/base-debian11@sha256:03dcbf61f859d0ae4c69c6242c9e5c3d7e1a42e5d3b69eb235e81a5810dd768e

COPY --from=builder /usr/bin/kas /usr/bin/kas

ENTRYPOINT ["/usr/bin/kas"]
