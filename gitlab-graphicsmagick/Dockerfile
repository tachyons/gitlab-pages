ARG FROM_IMAGE=debian
ARG TAG=stretch-slim

FROM ${FROM_IMAGE}:${TAG} as compile

ARG GM_VERSION=1.3.35
ARG _SIGNATURE_KEY=EBDFDB21B020EE8FD151A88DE301047DE1198975
ARG _SIGNATURE_ENFORCE=1

ENV GM_SOURCE_URL="https://downloads.sourceforge.net/project/graphicsmagick/graphicsmagick/${GM_VERSION}/GraphicsMagick-${GM_VERSION}.tar.xz"
ENV GM_SIGNATURE_URL="${SOURCE_URL}.sig"

RUN mkdir -p /build /target
COPY scripts/ scripts

RUN /scripts/dependencies
RUN /scripts/download
RUN /scripts/build

FROM scratch
COPY --from=compile /target/ /