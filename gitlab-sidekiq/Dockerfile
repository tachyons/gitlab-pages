ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG PYTHON_VERSION="3.7.3"
ARG TAG="master"
ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/gitlab-rails-ee"

FROM ${CI_REGISTRY_IMAGE}/gitlab-python:${PYTHON_VERSION} as python

FROM ${FROM_IMAGE}:${TAG}

ARG GITLAB_USER=git
ARG DOCUTILS_VERSION="0.16"

# BEGIN python install
COPY --from=python /usr/local/bin /usr/local/bin/
COPY --from=python /usr/local/lib /usr/local/lib/
ENV PYTHONPATH=/usr/local/lib/python3.7/site-packages
RUN ldconfig \
    && /usr/local/bin/pip3 install --compile docutils==${DOCUTILS_VERSION}
# END python install

ENV SIDEKIQ_CONCURRENCY=25
ENV SIDEKIQ_TIMEOUT=4

# Install runtime deps. openssh-client is required so that SSH client binaries
# are present for repository mirroring.
# xtail is necessary for logs that are tailed after the pod starts
# https://gitlab.com/gitlab-org/charts/gitlab/-/issues/1957
RUN apt-get update \
    && apt-get install -y --no-install-recommends openssh-client xtail

# Add scripts
COPY scripts/  /scripts/
RUN chown -R $GITLAB_USER:$GITLAB_USER /scripts

USER $GITLAB_USER:$GITLAB_USER

CMD /scripts/process-wrapper
