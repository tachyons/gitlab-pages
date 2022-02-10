ARG FROM_IMAGE=debian
ARG TAG=bullseye-slim

FROM ${FROM_IMAGE}:${TAG} AS download

ARG GOMPLATE_VERSION="v3.9.0"
ARG GOMPLATE_SHA256=7dbabe30095f822ec38f5f70711ff121c26e588227da4cc05208417cfaf929cd

ENV GOMPLATE_SOURCE_URL="https://github.com/hairyhenderson/gomplate/releases/download/$GOMPLATE_VERSION/gomplate_linux-amd64"

RUN apt-get update \
    && apt-get install -q -y --no-install-recommends ca-certificates curl

RUN mkdir -p /target
COPY scripts/download /scripts/download
RUN /scripts/download

FROM scratch
COPY --from=download /target/ /
