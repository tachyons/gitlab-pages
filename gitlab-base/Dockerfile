ARG DEBIAN_IMAGE=debian:bullseye-slim
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

# Configure default path for set-config
ENV CONFIG_TEMPLATE_DIRECTORY=/etc

ENTRYPOINT ["/scripts/entrypoint.sh"]