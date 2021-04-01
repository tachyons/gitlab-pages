ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG TAG="master"
ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/gitlab-rails-ee"
ARG PYTHON_TAG=3.9.2

FROM ${CI_REGISTRY_IMAGE}/gitlab-python:${PYTHON_TAG} AS python

FROM ${FROM_IMAGE}:${TAG}

ARG S3CMD_VERSION="2.1.0"
ARG GSUTIL_VERSION="4.43"

COPY --from=python /usr/local/bin /usr/local/bin/
COPY --from=python /usr/local/lib /usr/local/lib/
COPY --from=python /usr/local/include /usr/local/include/

RUN apt-get update \
	&& apt-get install -y --no-install-recommends \
  gcc \
  ca-certificates \
  openssl \
  tar \
  && ldconfig \
  && pip3 install --upgrade s3cmd==${S3CMD_VERSION} \
  && pip3 install gsutil==${GSUTIL_VERSION} crcmod \
  && apt-get purge -y --auto-remove gcc \
  && rm -rf /var/lib/apt/lists/*

ARG GITLAB_USER=git

COPY scripts/bin/* /usr/local/bin/
COPY scripts/lib/* /usr/lib/ruby/vendor_ruby/

USER $GITLAB_USER:$GITLAB_USER

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
