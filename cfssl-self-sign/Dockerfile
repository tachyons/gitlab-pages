# cfssl-self-sign
# 
# Container to auto-generate self-signed certificates for kubernetes
# - Generate self-signed CA
# - Generate a wild-card SSL certificate, based on that CA.
# - Expose all of the above through /output
ARG ALPINE_VERSION=3.10
FROM alpine:$ALPINE_VERSION

ARG CFSSL_VERSION="1.2"
ARG CFSSL_PLATFORM="linux-amd64"

COPY scripts/ /scripts/

RUN /scripts/install-cfssl.sh

VOLUME /output

CMD /scripts/generate-certificates
