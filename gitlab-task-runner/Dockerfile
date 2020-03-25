ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG TAG=latest
ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/gitlab-rails-ee"
ARG PYTHON_TAG=3.7.3

FROM ${CI_REGISTRY_IMAGE}/gitlab-python:${PYTHON_TAG} AS python

FROM ${FROM_IMAGE}:${TAG}

ARG S3CMD_VERSION="2.0.2"
ARG GSUTIL_VERSION="4.43"

COPY --from=python /usr/local/bin /usr/local/bin/
COPY --from=python /usr/local/lib /usr/local/lib/
COPY --from=python /usr/local/include /usr/local/include/

RUN apt-get update \
	&& apt-get install -y --no-install-recommends \
  ca-certificates \
  openssl \
  tar \
  patch \
  && ldconfig \
  && pip3 install --upgrade s3cmd==${S3CMD_VERSION} \
  && pip3 install gsutil==${GSUTIL_VERSION} crcmod \
  && rm -rf /var/lib/apt/lists/*

# Patch s3cmd for https://gitlab.com/gitlab-org/charts/gitlab/-/issues/1649
COPY patches/s3cmd-unicode.patch /tmp/
RUN cd $(pip3 show -f s3cmd | awk '$1 ~ /Location/ {print $NF}') \
    && patch -p1 < /tmp/s3cmd-unicode.patch \
    && rm /tmp/s3cmd-unicode.patch

ARG GITLAB_USER=git

COPY scripts/ /scripts

RUN cp scripts/bin/* /usr/local/bin && cp scripts/lib/* /usr/lib/ruby/vendor_ruby

USER $GITLAB_USER:$GITLAB_USER

ENTRYPOINT ["/scripts/bin/entrypoint.sh"]
