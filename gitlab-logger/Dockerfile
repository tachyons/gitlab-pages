ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/gitlab-go"
ARG GO_TAG="master"

FROM ${FROM_IMAGE}:${GO_TAG} as compile

ARG GITLAB_LOGGER_VERSION=1.0
ARG GITLAB_LOGGER_SHA256=f37acfd5cdbcc940f541bb2cde5468778db60c6eb4d2e5f368207cafefcdb66f

ENV GITLAB_LOGGER_SOURCE_URL="https://github.com/jbguerraz/gitlab-logger/archive/v${GITLAB_LOGGER_VERSION}.tar.gz"

RUN mkdir -p /build /target
COPY scripts/ /scripts

RUN /scripts/dependencies
RUN /scripts/download
RUN /scripts/build

FROM scratch
COPY --from=compile /target/ /
