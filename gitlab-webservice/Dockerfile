ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG PYTHON_TAG="3.8.12"
ARG TAG="master"
ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/gitlab-rails-ee"
ARG GITLAB_LOGGER_IMAGE="${CI_REGISTRY_IMAGE}/gitlab-logger:master"

FROM ${CI_REGISTRY_IMAGE}/gitlab-python:${PYTHON_TAG} as python
FROM ${GITLAB_LOGGER_IMAGE} as gitlab-logger

FROM ${FROM_IMAGE}:${TAG}

ARG DATADIR=/var/opt/gitlab
ARG CONFIG=/srv/gitlab/config
ARG DOCUTILS_VERSION="0.16"

COPY --from=python /usr/local/bin /usr/local/bin/
COPY --from=python /usr/local/lib /usr/local/lib/
ENV PYTHONPATH=/usr/local/lib/python3.8/site-packages
RUN ldconfig \
    && /usr/local/bin/pip3 install --compile docutils==${DOCUTILS_VERSION}
# END python install

# Pull in gitlab-logger
COPY --from=gitlab-logger /gitlab-logger /usr/local/bin/gitlab-logger

ARG GITLAB_USER=git

USER $GITLAB_USER:$GITLAB_USER

COPY scripts/ /scripts
COPY --chown=git configuration/ ${CONFIG}/

RUN cd /srv/gitlab && \
    mkdir -p public/uploads && chown ${GITLAB_USER}:${GITLAB_USER} public/uploads && \
    chmod 0700 public/uploads

ENV GITALY_FEATURE_DEFAULT_ON=1

CMD /scripts/process-wrapper

HEALTHCHECK --interval=30s --timeout=30s --retries=5 \
CMD /scripts/healthcheck
