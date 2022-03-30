FROM debian:bullseye-slim

ARG BUILD_DIR=/tmp/build
ARG RUBY_VERSION=2.7.5
ARG RUBYGEMS_VERSION=3.2.19
ARG BUNDLER_VERSION=2.3.6
ARG JEMALLOC_VERSION=5.2.1
ARG EXIFTOOL_VERSION=11.70

ENV LANG=C.UTF-8

# install runtime deps
RUN apt-get update \
  && apt-get install -y --no-install-recommends \
  bzip2 \
  ca-certificates \
  libffi-dev \
  libgdbm6 \
  libssl-dev \
  libyaml-dev \
  procps \
	busybox \
  zlib1g-dev \
  coreutils \
  curl \
  gnupg2 \
  && rm -rf /var/lib/apt/lists/* \
  && mkdir -p ${BUILD_DIR} \
  && ln -s /etc/ssl/certs/ca-certificates.crt /usr/lib/ssl/cert.pem

# skip installing gem documentation
RUN mkdir -p /usr/etc && \
    { echo 'install: --no-document';  echo 'update: --no-document'; } >> /usr/etc/gemrc

COPY shared/build-scripts/ /build-scripts
COPY patches/ ${BUILD_DIR}/patches

# Install Ruby from source
RUN buildDeps=' \
  autoconf \
  bison \
  dpkg-dev \
  gcc \
  libbz2-dev \
  libgdbm-dev \
  libglib2.0-dev \
  libncurses-dev \
  libxml2-dev \
  libxslt-dev \
  make \
  xz-utils' \
  && apt-get update \
  && apt-get install -y --no-install-recommends $buildDeps \
  && rm -rf /var/lib/apt/lists/* \
  && cd ${BUILD_DIR} \
  && curl --retry 6 -L -sfo jemalloc.tar.bz2 https://github.com/jemalloc/jemalloc/releases/download/${JEMALLOC_VERSION}/jemalloc-${JEMALLOC_VERSION}.tar.bz2 \
  && tar -xjf jemalloc.tar.bz2 \
  && cd jemalloc-${JEMALLOC_VERSION} \
  && ./autogen.sh --prefix=/usr --enable-prof \
  && make -j "$(nproc)" install \
  && cd .. \
  && curl -f --retry 6 https://gitlab.com/gitlab-org/build/omnibus-mirror/exiftool/-/archive/${EXIFTOOL_VERSION}/exiftool-${EXIFTOOL_VERSION}.tar.gz | tar -xz \
  && cd exiftool-${EXIFTOOL_VERSION} \
  && patch -p1 < ${BUILD_DIR}/patches/allow-only-tiff-jpeg-exif-strip.patch \
  && perl Makefile.PL \
  && make install \
  && cd .. \
  && export RUBY_MAJOR_VERSION="${RUBY_VERSION%.*}" \
  && curl --retry 6 -sfo ruby.tar.xz https://cache.ruby-lang.org/pub/ruby/${RUBY_MAJOR_VERSION}/ruby-${RUBY_VERSION}.tar.xz \
  && tar -xf ruby.tar.xz \
  && rm ruby.tar.xz \
  && cd ruby-${RUBY_VERSION} \
  && patch -p1 < ${BUILD_DIR}/patches/thread-memory-allocations-${RUBY_MAJOR_VERSION}.patch \
  && export LDFLAGS="-Wl,--no-as-needed" \
  && cflags="-fno-omit-frame-pointer" ./configure --prefix=/usr --with-jemalloc --with-out-ext=dbm,readline --enable-shared --disable-install-rdoc --disable-install-doc  --without-gmp --without-gdbm --without-tk --disable-dtrace \
  && make -j "$(nproc)" install \
  && cd \
  && apt-get purge -y --auto-remove $buildDeps \
  && rm -rf ${BUILD_DIR} \
  && gem update --system "$RUBYGEMS_VERSION" \
  && gem install bundler --version "$BUNDLER_VERSION" --force --no-document \
  && curl --retry 6 -sf https://codeload.github.com/ConnorAtherton/rb-readline/tar.gz/v0.5.5 | tar -xz \
  && ruby rb-readline-0.5.5/setup.rb && rm -rf rb-readline-0.5.5 \
	&& rm -rf /root/.gem/ \
	&& /build-scripts/cleanup-gems /usr/lib/ruby/gems

# Add scripts
COPY scripts/ /scripts

ENV CONFIG_TEMPLATE_DIRECTORY=/etc

ENTRYPOINT ["/scripts/entrypoint.sh"]

CMD [ "irb" ]
