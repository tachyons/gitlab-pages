# cfssl-self-sign
# 
# Container to auto-generate self-signed certificates for kubernetes
# - Generate self-signed CA
# - Generate a wild-card SSL certificate, based on that CA.
# - Expose all of the above through /output
ARG ALPINE_IMAGE=alpine:3.15
FROM $ALPINE_IMAGE as builder

ARG CFSSL_VERSION="1.6.1"
ARG CFSSL_PLATFORM="linux_amd64"
ARG CFSSL_BIN=/usr/local/bin

COPY scripts/install-cfssl.sh /scripts/install-cfssl.sh
# Add curl vs wget. Simplifies install-cfss.sh for many distros
RUN apk add --no-cache curl
RUN /scripts/install-cfssl.sh

FROM $ALPINE_IMAGE

ARG CFSSL_BIN=/usr/local/bin

COPY --from=builder ${CFSSL_BIN}/cfssl* ${CFSSL_BIN}/
COPY scripts/generate-certificates /scripts/generate-certificates

VOLUME /output

CMD /scripts/generate-certificates
