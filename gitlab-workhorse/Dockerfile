ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/git-base"
ARG GIT_TAG=2.26.0
ARG GITLAB_EDITION=gitlab-rails-ee
ARG RAILS_VERSION=latest
ARG RUBY_VERSION=latest

FROM ${FROM_IMAGE}:${GIT_TAG} as builder

ARG WORKHORSE_VERSION=v5.0.0
ARG BUILD_DIR=/tmp/build
ARG GITLAB_NAMESPACE="gitlab-org"
ARG FETCH_ARTIFACTS_PAT
ARG CI_API_V4_URL

RUN buildDeps=' \
  make' \
  && apt-get update \
  && apt-get install -y --no-install-recommends $buildDeps xtail \
  && mkdir -p ${BUILD_DIR} \
  && cd ${BUILD_DIR} \
  && echo "Downloading source code from ${CI_API_V4_URL}/projects/${GITLAB_NAMESPACE}%2Fgitlab-workhorse/repository/archive.tar.bz2?sha=${WORKHORSE_VERSION}" \
  && curl --header "PRIVATE-TOKEN: ${FETCH_ARTIFACTS_PAT}" -o workhorse.tar.bz2 "${CI_API_V4_URL}/projects/${GITLAB_NAMESPACE}%2Fgitlab-workhorse/repository/archive.tar.bz2?sha=${WORKHORSE_VERSION}" \
  && tar -xjf workhorse.tar.bz2 \
  && rm workhorse.tar.bz2 \
  && cd gitlab-workhorse-* \
  && make install \
  && cd \
  && rm -rf ${BUILD_DIR} \
  && apt-get purge -y --auto-remove $buildDeps \
  && rm -rf /var/lib/apt/lists/*

FROM ${CI_REGISTRY_IMAGE}/${GITLAB_EDITION}:${RAILS_VERSION} as rails

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

USER $GITLAB_USER:$GITLAB_USER

COPY scripts/ /scripts/

CMD /scripts/start-workhorse

HEALTHCHECK --interval=30s --timeout=30s --retries=5 \
CMD /scripts/healthcheck
