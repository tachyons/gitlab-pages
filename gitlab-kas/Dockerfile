ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG TAG="master"
ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/gitlab-go"

FROM ${FROM_IMAGE}:${TAG} as builder

ARG BAZELISK_VERSION=v1.11.0
ARG GITLAB_KAS_VERSION=v14.5.0
ARG GITLAB_NAMESPACE=gitlab-org
ARG BUILD_DIR=/tmp/build
ARG CI_SERVER_HOST
ARG CI_SERVER_PORT

# install debian packages
RUN apt-get update && apt-get install -y build-essential git

# install bazel
RUN go install github.com/bazelbuild/bazelisk@${BAZELISK_VERSION} \
    && ln -s /root/go/bin/bazelisk /root/go/bin/bazel
ENV PATH="${PATH}:/root/go/bin"

# install kas
RUN echo "Downloading source code" \
    && git clone "https://gitlab-token:${FETCH_ARTIFACTS_PAT}@gitlab.com:${CI_SERVER_PORT}/${GITLAB_NAMESPACE}/cluster-integration/gitlab-agent.git" --branch "${GITLAB_KAS_VERSION}" --single-branch "${BUILD_DIR}" \
    && cd ${BUILD_DIR} \
    && echo "Building kas binary" \
    && bazel run //cmd/kas:extract_kas -- /usr/bin

# build image is derived from debian 11, so we use that for run image
FROM gcr.io/distroless/base-debian11@sha256:03dcbf61f859d0ae4c69c6242c9e5c3d7e1a42e5d3b69eb235e81a5810dd768e

COPY --from=builder /usr/bin/kas /usr/bin/kas

ENTRYPOINT ["/usr/bin/kas"]
