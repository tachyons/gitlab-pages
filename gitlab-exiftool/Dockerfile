ARG GITLAB_BASE_IMAGE=

FROM ${GITLAB_BASE_IMAGE} as build

ARG BUILD_DIR=/tmp/build
ARG EXIFTOOL_VERSION=12.42

COPY patches/ ${BUILD_DIR}/patches

RUN apt-get update
RUN apt-get install -y --no-install-recommends \
         patch \
         perl \
         make

RUN curl -f --retry 6 https://gitlab.com/gitlab-org/build/omnibus-mirror/exiftool/-/archive/${EXIFTOOL_VERSION}/exiftool-${EXIFTOOL_VERSION}.tar.gz | tar -xz \
    && cd exiftool-${EXIFTOOL_VERSION} \
    && patch -p1 < ${BUILD_DIR}/patches/allow-only-tiff-jpeg-exif-strip.patch \
    && perl Makefile.PL \
    && mkdir -p /target \
    && make DESTDIR=/target install \
    && rm -Rf /target/usr/local/man

FROM scratch as final
COPY --from=build /target/ /
