ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/gitlab-go"
ARG GO_TAG="master"
ARG RUBY_IMAGE="${CI_REGISTRY_IMAGE}/gitlab-ruby:master"

ARG GOMPLATE_TAG=
ARG GOMPLATE_IMAGE="${CI_REGISTRY_IMAGE}/gitlab-gomplate:${GOMPLATE_TAG}"

FROM ${FROM_IMAGE}:${GO_TAG} as builder

ARG GITLAB_PAGES_VERSION=v1.30.0
ARG BUILD_DIR=/tmp/build
ARG GITLAB_NAMESPACE="gitlab-org"
ARG FETCH_ARTIFACTS_PAT
ARG CI_API_V4_URL="https://gitlab.com/api/v4"

RUN buildDeps=' \
  make' \
  && apt-get update \
  && apt-get install -y --no-install-recommends $buildDeps \
  && mkdir -p ${BUILD_DIR} \
  && cd ${BUILD_DIR} \
  && echo "Downloading source code from ${CI_API_V4_URL}/projects/${GITLAB_NAMESPACE}%2Fgitlab-pages/repository/archive.tar.bz2?sha=${GITLAB_PAGES_VERSION}" \
  && curl -f --retry 6 --header "PRIVATE-TOKEN: ${FETCH_ARTIFACTS_PAT}" -o pages.tar.bz2 "${CI_API_V4_URL}/projects/${GITLAB_NAMESPACE}%2Fgitlab-pages/repository/archive.tar.bz2?sha=${GITLAB_PAGES_VERSION}" \
  && tar -xjf pages.tar.bz2 \
  && rm pages.tar.bz2 \
  && cd gitlab-pages-* \
  && make 'gitlab-pages' \
  && install -m +x ./gitlab-pages /usr/local/bin/gitlab-pages \
  && cd \
  && rm -rf ${BUILD_DIR} \
  && apt-get purge -y --auto-remove $buildDeps \
  && rm -rf /var/lib/apt/lists/*

FROM ${GOMPLATE_IMAGE} as gomplate

FROM ${RUBY_IMAGE}

ARG GITLAB_USER=git
ARG CONFIG_DIRECTORY=/etc/gitlab-pages
ARG DATA_DIRECTORY=/srv/gitlab-pages

# create gitlab user, necessary directories and set ownership
RUN adduser --disabled-password --gecos 'GitLab' ${GITLAB_USER} \
    && mkdir -p ${CONFIG_DIRECTORY} ${DATA_DIRECTORY} /var/log/gitlab \
    && chown -R ${GITLAB_USER}:${GITLAB_USER} ${CONFIG_DIRECTORY} ${DATA_DIRECTORY} /var/log/gitlab

COPY --from=builder /usr/local/bin/gitlab-pages /bin/gitlab-pages

# Pull in gomplate
COPY --from=gomplate /gomplate /usr/local/bin/gomplate

ENV CONFIG_TEMPLATE_DIRECTORY=${CONFIG_DIRECTORY}

USER $GITLAB_USER:$GITLAB_USER

COPY scripts/ /scripts/

CMD ["/scripts/start-pages"]

HEALTHCHECK --interval=30s --timeout=30s --retries=5 \
CMD /scripts/healthcheck
