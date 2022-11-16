ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/gitlab-go"
ARG GO_TAG="master"

FROM ${FROM_IMAGE}:${GO_TAG} as compile

ARG GITLAB_LOGGER_VERSION=v2.1.0
ARG GITLAB_LOGGER_SHA256=d7b7af60dbe28ecf2a9b84e6dac7d340d1aa9b0eff23914f0175d80ade7317b0

ENV GITLAB_LOGGER_SOURCE_URL="https://gitlab.com/gitlab-org/charts/components/gitlab-logger/-/archive/${GITLAB_LOGGER_VERSION}/gitlab-logger-${GITLAB_LOGGER_VERSION}.tar.gz"

RUN mkdir -p /build /target
COPY scripts/ /scripts

RUN /scripts/dependencies
RUN /scripts/download
RUN /scripts/build

FROM scratch
COPY --from=compile /target/ /
