ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG TAG="master"
ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/gitlab-go"
ARG RUBY_IMAGE="$CI_REGISTRY_IMAGE/gitlab-ruby"
ARG GITLAB_LOGGER_IMAGE="${CI_REGISTRY_IMAGE}/gitlab-logger"
ARG GITLAB_LOGGER_TAG="${GITLAB_LOGGER_TAG}"

FROM ${GITLAB_LOGGER_IMAGE}:${GITLAB_LOGGER_TAG} as logger
FROM ${FROM_IMAGE}:${TAG} as builder

ARG GITALY_SERVER_VERSION=v1.83.0
ARG BUILD_DIR=/tmp/build
ARG GITLAB_NAMESPACE="gitlab-org"
ARG FETCH_ARTIFACTS_PAT
ARG CI_API_V4_URL
ARG BUNDLE_OPTIONS="--jobs 4"

# install runtime deps. openssh-client is required so that SSH client binaries
# are present for repository mirroring.
RUN apt-get update \
    && apt-get install -y --no-install-recommends libicu67 net-tools openssh-client xtail

COPY shared/build-scripts/ /build-scripts

ARG CACHE_BUSTER=false

# Install build dependencies required to build Git, libgit2 and Ruby Gems.
RUN buildDeps=' \
    make \
    cmake \
    gcc \
    g++ \
    git \
    pkg-config' && \
    runDeps=' \
    libicu-dev \
    libpcre2-dev \
    libcurl4-gnutls-dev' && \
    apt-get update && \
    apt-get install -y --no-install-recommends $buildDeps $runDeps && \
    mkdir -p ${BUILD_DIR} && \
    cd ${BUILD_DIR} && \
    echo "Downloading source code from ${CI_API_V4_URL}/projects/${GITLAB_NAMESPACE}%2Fgitaly/repository/archive.tar.bz2?sha=${GITALY_SERVER_VERSION}" && \
    curl -f --retry 6 --header "PRIVATE-TOKEN: ${FETCH_ARTIFACTS_PAT}" -o gitaly.tar.bz2 "${CI_API_V4_URL}/projects/${GITLAB_NAMESPACE}%2Fgitaly/repository/archive.tar.bz2?sha=${GITALY_SERVER_VERSION}" && \
    tar -xjf gitaly.tar.bz2 --strip-components=1 && \
    rm gitaly.tar.bz2 && \
    cd ruby && \
    bundle config set --local deployment 'true' && \
    bundle config set --local without 'development test' && \
    bundle install ${BUNDLE_OPTIONS} && \
    cd .. && \
    touch .ruby-bundle && \
    make install WITH_BUNDLED_GIT=YesPlease && \
    rm -rf ${BUILD_DIR}/ruby/spec ${BUILD_DIR}/ruby/features && \
    mv ${BUILD_DIR}/ruby /srv/gitaly-ruby && \
    /build-scripts/cleanup-gems /srv/gitaly-ruby/vendor/bundle/ruby && \
    mkdir /target && \
    cp -R --parents \
      /usr/local/bin/gitaly* \
      /usr/local/bin/praefect \
      /usr/lib/ruby/gems/ \
      /srv/gitaly-ruby \
      /target

# Final image

FROM ${RUBY_IMAGE}


ARG GITLAB_USER=git

# install runtime deps. openssh-client is required so that SSH client binaries
# are present for repository mirroring.
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
      libpcre2-16-0 libpcre2-32-0 libpcre2-posix2 libcurl3-gnutls \
      libicu67 net-tools openssh-client xtail \
    && rm -rf /var/lib/apt/lists/*

# create gitlab user
# openssh daemon does not allow locked user to login, change ! to *
RUN adduser --disabled-password --gecos 'GitLab' ${GITLAB_USER} && \
      sed -i "s/${GITLAB_USER}:!/${GITLAB_USER}:*/" /etc/shadow

# configure runtime environment, with permissions
RUN install -d -o ${GITLAB_USER} -g ${GITLAB_USER} -m 755 /home/${GITLAB_USER}/repositories && \
    install -d -o ${GITLAB_USER} -m 755 /etc/gitaly && \
    install -o ${GITLAB_USER} -g ${GITLAB_USER} -m 0640 /dev/null /etc/gitaly/config.toml && \
    install -d -o ${GITLAB_USER} -m 755 /srv/gitlab-shell && \
    install -d -o ${GITLAB_USER} -m 755 /var/log/gitaly && \
    install -o ${GITLAB_USER} -g ${GITLAB_USER} -m 0644 /dev/null /var/log/gitaly/gitaly.log && \
    install -o ${GITLAB_USER} -g ${GITLAB_USER} -m 0644 /dev/null /var/log/gitaly/gitlab-shell.log

# add gitlab-logger
COPY --from=logger /gitlab-logger /usr/local/bin

# pull contents from builder
COPY --from=builder /target/ /
RUN cd /srv/gitaly-ruby && bundle binstubs --all

# Add scripts
COPY scripts/ /scripts/
COPY config.toml /etc/gitaly/config.toml

# ensure explicit permissions on scripts
RUN chmod -R =rx /scripts/

# set runtime user
USER ${GITLAB_USER}:${GITLAB_USER}

ENV CONFIG_TEMPLATE_DIRECTORY=/etc/gitaly
ENV PATH "$PATH:/srv/gitaly-ruby/bin"

CMD ["/scripts/process-wrapper"]

VOLUME /var/log/gitaly

HEALTHCHECK --interval=30s --timeout=10s --retries=5 \
CMD /scripts/healthcheck
