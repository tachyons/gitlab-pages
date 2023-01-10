ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/gitlab-go"
ARG GO_TAG="master"

FROM ${FROM_IMAGE}:${GO_TAG} as compile

ARG GITLAB_LOGGER_VERSION=v2.2.1
ARG GITLAB_LOGGER_SHA256=27bfa6f3859d142dde721b875da39b649c674d58f5c1bf4bd3aea0882bbf0e10

ENV GITLAB_LOGGER_SOURCE_URL="https://gitlab.com/gitlab-org/cloud-native/gitlab-logger/-/archive/${GITLAB_LOGGER_VERSION}/gitlab-logger-${GITLAB_LOGGER_VERSION}.tar.gz"

RUN mkdir -p /build /target
COPY scripts/ /scripts

RUN /scripts/dependencies
RUN /scripts/download
RUN /scripts/build

FROM scratch
COPY --from=compile /target/ /
