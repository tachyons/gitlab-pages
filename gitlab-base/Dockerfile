ARG DEBIAN_IMAGE=debian:bullseye-slim
ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"

ARG GOMPLATE_TAG=master
ARG GOMPLATE_IMAGE="${CI_REGISTRY_IMAGE}/gitlab-gomplate:${GOMPLATE_TAG}"

ARG GITLAB_LOGGER_IMAGE="${CI_REGISTRY_IMAGE}/gitlab-logger:master"

FROM ${GOMPLATE_IMAGE} as gomplate

ARG DEBIAN_IMAGE
ARG CI_REGISTRY_IMAGE
ARG GITLAB_LOGGER_IMAGE

FROM ${GITLAB_LOGGER_IMAGE} as gitlab-logger

ARG DEBIAN_IMAGE

FROM ${DEBIAN_IMAGE}

# Set default LANG
ENV LANG=C.UTF-8

# install shared runtime dependencies
RUN apt-get update \
  && apt-get install -y --no-install-recommends \
    procps \
    busybox \
    less \
    xtail \
    curl \
    ca-certificates \
  && ln -s /etc/ssl/certs/ca-certificates.crt /usr/lib/ssl/cert.pem

# Add scripts
COPY scripts/ /scripts
RUN chown -R 0:0 /scripts/

# Install gomplate
COPY --from=gomplate /gomplate /usr/local/bin/gomplate

# Install gitlab-logger
COPY --from=gitlab-logger /gitlab-logger /usr/local/bin/gitlab-logger

# Configure default path for set-config
ENV CONFIG_TEMPLATE_DIRECTORY=/etc

ENTRYPOINT ["/scripts/entrypoint.sh"]
