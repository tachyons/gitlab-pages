ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/git-base"
ARG GIT_TAG=2.24.1

FROM ${FROM_IMAGE}:${GIT_TAG} as builder

ARG DOCKER_BUILDTAGS="include_oss include_gcs"
ARG REGISTRY_VERSION=v2.8.0-gitlab
ARG REGISTRY_NAMESPACE=gitlab-org
ARG REGISTRY_PROJECT=container-registry
ARG GOPATH=/go
ARG REGISTRY_SOURCE_PATH=${GOPATH}/src/github.com/docker/distribution

RUN buildDeps=' \
  make' \
  && apt-get update \
  && apt-get install -y --no-install-recommends $buildDeps \
  && git clone https://gitlab.com/${REGISTRY_NAMESPACE}/${REGISTRY_PROJECT}.git \
    --branch ${REGISTRY_VERSION} --single-branch ${REGISTRY_SOURCE_PATH} \
  && cd ${REGISTRY_SOURCE_PATH} \
  && CGO_ENABLED=0 BUILDTAGS=${DOCKER_BUILDTAGS} make clean binaries \
  && cp bin/registry /usr/local/bin/registry \
  && rm -rf ${GOPATH} \
  && apt-get purge -y --auto-remove $buildDeps \
  && rm -rf /var/lib/apt/lists/*

FROM debian:stretch-slim

ARG GITLAB_USER=git

# create gitlab user
RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates apache2-utils \
  && rm -rf /var/lib/apt/lists/* \
  && adduser --disabled-password --gecos 'GitLab' ${GITLAB_USER}

COPY --from=builder /usr/local/bin/registry /bin/registry

USER $GITLAB_USER:$GITLAB_USER

ENV CONFIG_DIRECTORY=/etc/docker/registry
ENV CONFIG_FILENAME=config.yml

CMD /bin/registry serve ${CONFIG_DIRECTORY}/${CONFIG_FILENAME}
