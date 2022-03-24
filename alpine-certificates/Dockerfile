# alpine-certificates:
# - Alpine, with ca-certificates installed. We can't assume that we'll be able
#   operate as root, or access the internet to install the package.
# - This image is intended to be based on alpine:latest, which is currently 3.15.
#   We're doing this to ensure the latest ca-certificates package.
ARG FROM_IMAGE=alpine
ARG ALPINE_VERSION=3.15
FROM $FROM_IMAGE:$ALPINE_VERSION

ARG CA_PKG_VERSION=20211220-r0
RUN apk --update-cache --no-cache add ca-certificates=${CA_PKG_VERSION}

COPY scripts/bundle-certificates /scripts/

VOLUME /etc/ssl/certs /usr/local/share/ca-certificates

CMD ["/scripts/bundle-certificates"]
