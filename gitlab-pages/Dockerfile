ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG TAG="master"

ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/gitlab-go"
ARG GO_TAG="${TAG}"
ARG GITLAB_BASE_IMAGE="${CI_REGISTRY_IMAGE}/gitlab-base:${TAG}"

FROM cgr.dev/chainguard/go:1.19-dev as builder

ARG GITLAB_PAGES_VERSION=v1.30.0
ARG BUILD_DIR=/tmp/build
ARG GITLAB_NAMESPACE="gitlab-org"
ARG FETCH_ARTIFACTS_PAT
ARG CI_API_V4_URL="https://gitlab.com/api/v4"

RUN apk add curl \
  && mkdir -p ${BUILD_DIR} \
  && cd ${BUILD_DIR} \
  && echo "Downloading source code from ${CI_API_V4_URL}/projects/${GITLAB_NAMESPACE}%2Fgitlab-pages/repository/archive.tar.bz2?sha=${GITLAB_PAGES_VERSION}" \
  && curl -f --retry 6 --header "PRIVATE-TOKEN: ${FETCH_ARTIFACTS_PAT}" -o pages.tar.bz2 "${CI_API_V4_URL}/projects/${GITLAB_NAMESPACE}%2Fgitlab-pages/repository/archive.tar.bz2?sha=${GITLAB_PAGES_VERSION}" \
  && tar -xjf pages.tar.bz2 \
  && rm pages.tar.bz2 \
  && cd gitlab-pages-* \
  && make 'gitlab-pages' "LAST_TAG=v${GITLAB_PAGES_VERSION}" "VERSION=${GITLAB_PAGES_VERSION}" \
  && install -m +x ./gitlab-pages /usr/bin/gitlab-pages

## FINAL IMAGE ##

FROM cgr.dev/chainguard/wolfi-base:latest

ARG GITLAB_USER=git
ARG CONFIG_DIRECTORY=/etc/gitlab-pages
ARG DATA_DIRECTORY=/srv/gitlab-pages

# create gitlab user, necessary directories and set ownership
RUN adduser --disabled-password --gecos 'GitLab' ${GITLAB_USER} \
    && mkdir -p ${CONFIG_DIRECTORY} ${DATA_DIRECTORY} /var/log/gitlab \
    && chown -R ${GITLAB_USER}:${GITLAB_USER} ${CONFIG_DIRECTORY} ${DATA_DIRECTORY} /var/log/gitlab

COPY --from=builder /usr/bin/gitlab-pages /usr/bin/gitlab-pages

ENV CONFIG_TEMPLATE_DIRECTORY=${CONFIG_DIRECTORY}

USER $GITLAB_USER:$GITLAB_USER

COPY scripts/ /scripts/

CMD ["/scripts/start-pages"]

HEALTHCHECK --interval=30s --timeout=30s --retries=5 \
CMD /scripts/healthcheck
