ARG DEBIAN_IMAGE=debian:bullseye-slim
FROM ${DEBIAN_IMAGE} AS download

ARG GOMPLATE_VERSION="v3.11.3"
ARG GOMPLATE_SHA256=2c67ef580d2416e82dc2ab8f0c30e1bf372dcd3e8511073df9310d91dc3d5f94

ENV GOMPLATE_SOURCE_URL="https://github.com/hairyhenderson/gomplate/releases/download/$GOMPLATE_VERSION/gomplate_linux-amd64"

RUN apt-get update \
    && apt-get install -q -y --no-install-recommends ca-certificates curl

RUN mkdir -p /target
COPY scripts/download /scripts/download
RUN /scripts/download

FROM scratch
COPY --from=download /target/ /
