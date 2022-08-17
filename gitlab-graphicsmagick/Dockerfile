ARG DEBIAN_IMAGE=debian:bullseye-slim
FROM ${DEBIAN_IMAGE} as compile

ARG GM_VERSION_MINOR=1.3
ARG GM_VERSION=1.3.36
ARG GM_CHECKSUM_SHA256
ARG _SIGNATURE_KEY=EBDFDB21B020EE8FD151A88DE301047DE1198975
ARG _SIGNATURE_ENFORCE=1

ENV GM_SOURCE_URL="https://sourceforge.net/projects/graphicsmagick/files/graphicsmagick/${GM_VERSION}/GraphicsMagick-${GM_VERSION}.tar.xz/download"
ENV GM_SIGNATURE_URL="https://sourceforge.net/projects/graphicsmagick/files/graphicsmagick/${GM_VERSION}/GraphicsMagick-${GM_VERSION}.tar.xz.sig/download"

RUN mkdir -p /build /target
COPY scripts/ /scripts

RUN /scripts/dependencies
RUN /scripts/download
RUN /scripts/build

FROM scratch
COPY --from=compile /target/ /
