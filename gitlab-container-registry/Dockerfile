ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/git-base"
ARG GIT_TAG=2.22.0

FROM ${FROM_IMAGE}:${GIT_TAG} as builder

ARG REGISTRY_VERSION=v2.7.1
ARG REGISTRY_NAMESPACE=gitlab-org
ARG REGISTRY_PROJECT=container-registry
ARG GOPATH=/go

RUN buildDeps=' \
  make' \
  && apt-get update \
  && apt-get install -y --no-install-recommends $buildDeps \
  && GOPATH=${GOPATH} go get gitlab.com/${REGISTRY_NAMESPACE}/${REGISTRY_PROJECT} \
  && cd ${GOPATH}/src/gitlab.com/${REGISTRY_NAMESPACE}/${REGISTRY_PROJECT} \
  && git checkout ${REGISTRY_VERSION} \
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
