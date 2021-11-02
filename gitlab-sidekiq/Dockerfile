ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG PYTHON_TAG="3.8.9"
ARG TAG="master"
ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/gitlab-rails-ee"
ARG GITLAB_LOGGER_IMAGE="${CI_REGISTRY_IMAGE}/gitlab-logger:master"

FROM ${CI_REGISTRY_IMAGE}/gitlab-python:${PYTHON_TAG} as python
FROM ${GITLAB_LOGGER_IMAGE} as gitlab-logger

FROM ${FROM_IMAGE}:${TAG}

ARG GITLAB_USER=git
ARG DOCUTILS_VERSION="0.16"

# BEGIN python install
COPY --from=python /usr/local/bin /usr/local/bin/
COPY --from=python /usr/local/lib /usr/local/lib/
ENV PYTHONPATH=/usr/local/lib/python3.8/site-packages
RUN ldconfig \
    && /usr/local/bin/pip3 install --compile docutils==${DOCUTILS_VERSION}
# END python install

ENV SIDEKIQ_CONCURRENCY=25
ENV SIDEKIQ_TIMEOUT=25

# Install runtime deps. openssh-client is required so that SSH client binaries
# are present for repository mirroring.
# xtail is necessary for logs that are tailed after the pod starts
# https://gitlab.com/gitlab-org/charts/gitlab/-/issues/1957
RUN apt-get update \
    && apt-get install -y --no-install-recommends openssh-client xtail

# Add scripts
COPY scripts/  /scripts/

# Pull in gitlab-logger
COPY --from=gitlab-logger /gitlab-logger /usr/local/bin/gitlab-logger

RUN chown -R $GITLAB_USER:$GITLAB_USER /scripts

USER $GITLAB_USER:$GITLAB_USER

CMD /scripts/process-wrapper
