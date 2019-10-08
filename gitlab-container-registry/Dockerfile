ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/git-base"
ARG GIT_TAG=2.22.0
ARG RUBY_VERSION=latest

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

ARG DATADIR=/var/opt/gitlab
ARG CONFIG=/srv/gitlab/config
ARG GITLAB_USER=git

# create gitlab user
RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates \
  && rm -rf /var/lib/apt/lists/* \
  && adduser --disabled-password --gecos 'GitLab' ${GITLAB_USER} \
  && install -d -o ${GITLAB_USER} /var/log/gitlab/ \
  && install -d -o ${GITLAB_USER} ${DATADIR} \
  && install -d -o ${GITLAB_USER} ${CONFIG}

COPY --from=builder /usr/local/bin/registry /usr/local/bin/registry

USER $GITLAB_USER:$GITLAB_USER

COPY scripts/ /scripts/

CMD /scripts/start-registry

HEALTHCHECK --interval=30s --timeout=30s --retries=5 \
CMD /scripts/healthcheck
