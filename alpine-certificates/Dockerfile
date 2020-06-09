# alpine-certificates:
# - Alpine, with ca-certificates installed. We can't assume that we'll be able
#   operate as root, or access the internet to install the package.
# - This image is intended to be based on alpine:latest, which is currently 3.10.
#   We're doing this to ensure the latest ca-certificates package.
ARG ALPINE_VERSION=3.10
FROM alpine:$ALPINE_VERSION

ARG CA_PKG_VERSION=20191127-r3
RUN apk --update --no-cache add ca-certificates=${CA_PKG_VERSION}

COPY scripts/ /scripts/

VOLUME /etc/ssl/certs /usr/local/share/ca-certificates

CMD ["/scripts/bundle-certificates"]
