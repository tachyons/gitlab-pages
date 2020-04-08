FROM debian:stretch-slim

ARG BUILD_DIR=/tmp/build
ARG RUBY_MAJOR_VERSION=2.6
ARG RUBY_MINOR_VERSION=2.6.5
ARG RUBYGEMS_VERSION=2.7.10
ARG BUNDLER_VERSION=1.17.3

ENV LANG=C.UTF-8

# install runtime deps
RUN apt-get update \
	&& apt-get install -y --no-install-recommends \
  bzip2 \
  ca-certificates \
  libffi-dev \
  libgdbm3 \
  libssl1.0-dev \
  libyaml-dev \
  procps \
	busybox \
  zlib1g-dev \
  coreutils \
  curl \
  gnupg2 \
  libimage-exiftool-perl \
  && rm -rf /var/lib/apt/lists/* \
  && mkdir -p ${BUILD_DIR}

# skip installing gem documentation
RUN mkdir -p /usr/etc && \
    { echo 'install: --no-document';  echo 'update: --no-document'; } >> /usr/etc/gemrc

COPY shared/build-scripts/ /build-scripts

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
  && curl --retry 6 -so ruby.tar.xz https://cache.ruby-lang.org/pub/ruby/${RUBY_MAJOR_VERSION}/ruby-${RUBY_MINOR_VERSION}.tar.xz \
  && tar -xf ruby.tar.xz \
  && rm ruby.tar.xz \
  && cd ruby-${RUBY_MINOR_VERSION} \
  && ./configure --prefix=/usr --with-out-ext=dbm,readline --enable-shared --disable-install-rdoc --disable-install-doc  --without-gmp --without-gdbm --without-tk --disable-dtrace \
  && make -j "$(nproc)" \
  && make -j "$(nproc)" install \
  && cd \
  && apt-get purge -y --auto-remove $buildDeps \
  && rm -rf ${BUILD_DIR} \
  && gem update --system "$RUBYGEMS_VERSION" \
  && gem install bundler --version "$BUNDLER_VERSION" --force --no-ri --no-rdoc \
  && curl --retry 6 -s https://codeload.github.com/ConnorAtherton/rb-readline/tar.gz/v0.5.5 | tar -xz \
  && ruby rb-readline-0.5.5/setup.rb && rm -rf rb-readline-0.5.5 \
	&& rm -rf /root/.gem/ \
	&& /build-scripts/cleanup-gems /usr/lib/ruby/gems

# Add scripts
COPY scripts/ /scripts

ENV CONFIG_TEMPLATE_DIRECTORY=/etc

ENTRYPOINT ["/scripts/entrypoint.sh"]

CMD [ "irb" ]
