ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/gitlab-go"
ARG GO_TAG="master"

FROM ${FROM_IMAGE}:${GO_TAG} as compile

ARG GITLAB_LOGGER_VERSION=v2.2.0
ARG GITLAB_LOGGER_SHA256=0d118cbbb721d884c6541da64adb3277900f3d46047473eeefcf93420d2a3a8d

ENV GITLAB_LOGGER_SOURCE_URL="https://gitlab.com/gitlab-org/charts/components/gitlab-logger/-/archive/${GITLAB_LOGGER_VERSION}/gitlab-logger-${GITLAB_LOGGER_VERSION}.tar.gz"

RUN mkdir -p /build /target
COPY scripts/ /scripts

RUN /scripts/dependencies
RUN /scripts/download
RUN /scripts/build

FROM scratch
COPY --from=compile /target/ /
