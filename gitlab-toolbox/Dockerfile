ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG TAG="master"
ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/gitlab-rails-ee"
ARG PYTHON_TAG=3.8.9
ARG GITALY_IMAGE=

FROM ${CI_REGISTRY_IMAGE}/gitlab-python:${PYTHON_TAG} AS python

FROM ${GITALY_IMAGE} AS gitaly

FROM ${FROM_IMAGE}:${TAG}

ARG AWSCLI_VERSION="1.20.44"
ARG S3CMD_VERSION="2.1.0"
ARG GSUTIL_VERSION="4.43"

COPY --from=python /usr/local/bin /usr/local/bin/
COPY --from=python /usr/local/lib /usr/local/lib/
COPY --from=python /usr/local/include /usr/local/include/
COPY --from=gitaly  /usr/local/bin/gitaly-backup /usr/local/bin/

RUN apt-get update \
	&& apt-get install -y --no-install-recommends \
  gcc \
  ca-certificates \
  openssl \
  tar \
  && ldconfig \
  && pip3 install --upgrade awscli==${AWSCLI_VERSION} s3cmd==${S3CMD_VERSION} \
  && pip3 install gsutil==${GSUTIL_VERSION} crcmod \
  && pip3 cache purge \
  && find /usr/local/lib/python3.8 -name '__pycache__' -type d -print -exec rm -r {} + \
  && apt-get purge -y --auto-remove gcc \
  && rm -rf /var/lib/apt/lists/*

ARG GITLAB_USER=git

COPY scripts/bin/* /usr/local/bin/
COPY scripts/lib/* /usr/lib/ruby/vendor_ruby/

USER $GITLAB_USER:$GITLAB_USER

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
