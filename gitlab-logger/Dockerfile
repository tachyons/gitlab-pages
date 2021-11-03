ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/gitlab-go"
ARG GO_TAG="master"

FROM ${FROM_IMAGE}:${GO_TAG} as compile

ARG GITLAB_LOGGER_VERSION=v1.3.1
ARG GITLAB_LOGGER_SHA256=1f53389f9efe306d0b98faa7b75c312965baf46831f3d21b94b4e5d7cd133329

ENV GITLAB_LOGGER_SOURCE_URL="https://gitlab.com/gitlab-org/charts/components/gitlab-logger/-/archive/${GITLAB_LOGGER_VERSION}/gitlab-logger-${GITLAB_LOGGER_VERSION}.tar.gz"

RUN mkdir -p /build /target
COPY scripts/ /scripts

RUN /scripts/dependencies
RUN /scripts/download
RUN /scripts/build

FROM scratch
COPY --from=compile /target/ /
