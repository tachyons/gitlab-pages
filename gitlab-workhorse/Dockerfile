ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG GIT_TAG=2.29.0
ARG GITLAB_EDITION=gitlab-rails-ee
ARG RAILS_VERSION="master"
ARG RUBY_VERSION="master"

ARG GOMPLATE_TAG=
ARG GOMPLATE_IMAGE="${CI_REGISTRY_IMAGE}/gitlab-gomplate:${GOMPLATE_TAG}"

FROM ${GOMPLATE_IMAGE} as gomplate
FROM ${CI_REGISTRY_IMAGE}/${GITLAB_EDITION}:${RAILS_VERSION} as rails
FROM ${CI_REGISTRY_IMAGE}/git-base:${GIT_TAG} as builder

ARG BUILD_DIR=/tmp/build
RUN mkdir -p ${BUILD_DIR}
COPY --from=rails /srv/gitlab/ ${BUILD_DIR}/

RUN buildDeps=' \
  make' \
  && apt-get update \
  && apt-get install -y --no-install-recommends $buildDeps \
  && make -C ${BUILD_DIR}/workhorse install \
  && rm -rf ${BUILD_DIR} \
  && apt-get purge -y --auto-remove $buildDeps \
  && rm -rf /var/lib/apt/lists/*

FROM ${CI_REGISTRY_IMAGE}/gitlab-ruby:${RUBY_VERSION}

ARG DATADIR=/var/opt/gitlab
ARG CONFIG=/srv/gitlab/config
ARG GITLAB_USER=git

# create gitlab user
RUN adduser --disabled-password --gecos 'GitLab' ${GITLAB_USER} && \
  install -d -o ${GITLAB_USER} /var/log/gitlab/ && \
  install -d -o ${GITLAB_USER} ${DATADIR} && \
  install -d -o ${GITLAB_USER} ${CONFIG}

COPY --from=builder /usr/local/bin/gitlab-* /usr/local/bin/

COPY --from=rails --chown=git /srv/gitlab/public/ /srv/gitlab/public/

COPY --from=rails --chown=git /srv/gitlab/doc/ /srv/gitlab/doc/

COPY --from=gomplate /gomplate /usr/local/bin/gomplate

USER $GITLAB_USER:$GITLAB_USER

COPY scripts/ /scripts/

CMD /scripts/start-workhorse

HEALTHCHECK --interval=30s --timeout=30s --retries=5 \
CMD /scripts/healthcheck
