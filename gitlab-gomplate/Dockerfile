ARG DEBIAN_IMAGE=debian:bullseye-slim
FROM ${DEBIAN_IMAGE} AS download

ARG GOMPLATE_VERSION="v3.11.1"
ARG GOMPLATE_SHA256=e3d3c32075c6523bd6d0a999977d1a9619b7d39549381b40dad8a82c963fc13e

ENV GOMPLATE_SOURCE_URL="https://github.com/hairyhenderson/gomplate/releases/download/$GOMPLATE_VERSION/gomplate_linux-amd64"

RUN apt-get update \
    && apt-get install -q -y --no-install-recommends ca-certificates curl

RUN mkdir -p /target
COPY scripts/download /scripts/download
RUN /scripts/download

FROM scratch
COPY --from=download /target/ /
