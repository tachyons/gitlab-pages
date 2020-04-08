FROM debian:stretch-slim

ARG BUILD_DIR=/tmp/build
ARG PG_VERSION=11.7

ENV LANG=C.UTF-8

# install runtime deps
RUN apt-get update \
	&& apt-get install -y --no-install-recommends \
  bzip2 \
  ca-certificates \
  libffi-dev \
  libssl1.0-dev \
  libyaml-dev \
  busybox \
  zlib1g-dev \
  coreutils \
  libossp-uuid16 \
  curl \
  gnupg2 \
  && rm -rf /var/lib/apt/lists/* \
  && mkdir -p ${BUILD_DIR}

# Install Postgresql from source
RUN buildDeps=' \
  autoconf \
  bison \
  gcc \
  libbz2-dev \
  libglib2.0-dev \
  libncurses-dev \
  libedit-dev \
  libxml2-dev \
  libossp-uuid-dev \
  libxslt-dev \
  make' \
  && apt-get update \
  && apt-get install -y --no-install-recommends $buildDeps \
  && rm -rf /var/lib/apt/lists/* \
  && cd ${BUILD_DIR} \
  && curl --retry 6 -so postgresql.tar.gz https://ftp.postgresql.org/pub/source/v${PG_VERSION}/postgresql-${PG_VERSION}.tar.gz \
  && tar -xzf postgresql.tar.gz \
  && rm postgresql.tar.gz \
  && cd postgresql-${PG_VERSION} \
  && ./configure --prefix=/usr/local/psql --with-libedit-preferred --disable-rpath --with-openssl --with-uuid=ossp \
  && make -j "$(nproc)" world \
  && make -j "$(nproc)" install-world \
  && cd \
  && apt-get purge -y --auto-remove $buildDeps \
  && rm -rf ${BUILD_DIR}
