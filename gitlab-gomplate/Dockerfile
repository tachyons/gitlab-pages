ARG DEBIAN_IMAGE=debian:bullseye-slim
FROM ${DEBIAN_IMAGE} AS download

ARG GOMPLATE_VERSION="v3.11.4"
ARG GOMPLATE_SHA256=f69d38e101c5524e7aad43b90a5c06b34363ce7891e3c37a496d70d3660e819c

ENV GOMPLATE_SOURCE_URL="https://github.com/hairyhenderson/gomplate/releases/download/$GOMPLATE_VERSION/gomplate_linux-amd64"

RUN apt-get update \
    && apt-get install -q -y --no-install-recommends ca-certificates curl

RUN mkdir -p /target
COPY scripts/download /scripts/download
RUN /scripts/download

FROM scratch
COPY --from=download /target/ /
