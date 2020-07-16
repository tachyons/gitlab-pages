ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG TAG=latest
ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/git-base"

FROM ${FROM_IMAGE}:${TAG}

ARG GITLAB_USER=git
ARG GITALY_SERVER_VERSION=v1.83.0
ARG BUILD_DIR=/tmp/build
ARG GITLAB_NAMESPACE="gitlab-org"
ARG FETCH_ARTIFACTS_PAT
ARG CI_API_V4_URL
ARG BUNDLE_OPTIONS="--jobs 4 --without development test"

# install runtime deps. openssh-client is required so that SSH client binaries
# are present for repository mirroring.
RUN apt-get update \
    && apt-get install -y --no-install-recommends libicu57 net-tools openssh-client xtail

# create gitlab user
# openssh daemon does not allow locked user to login, change ! to *
RUN adduser --disabled-password --gecos 'GitLab' ${GITLAB_USER} && \
      sed -i "s/${GITLAB_USER}:!/${GITLAB_USER}:*/" /etc/shadow

# create default repository directory
RUN sudo -u ${GITLAB_USER} -H mkdir -p /home/${GITLAB_USER}/repositories

COPY shared/build-scripts/ /build-scripts

# Download and compile Gitaly
ARG CACHE_BUSTER=false
RUN cd ${BUILD_DIR} && \
    echo "Downloading source code from ${CI_API_V4_URL}/projects/${GITLAB_NAMESPACE}%2Fgitaly/repository/archive.tar.bz2?sha=${GITALY_SERVER_VERSION}" && \
    curl --retry 6 --header "PRIVATE-TOKEN: ${FETCH_ARTIFACTS_PAT}" -o gitaly.tar.bz2 "${CI_API_V4_URL}/projects/${GITLAB_NAMESPACE}%2Fgitaly/repository/archive.tar.bz2?sha=${GITALY_SERVER_VERSION}" && \
    tar -xjf gitaly.tar.bz2 --strip-components=1 && \
    rm gitaly.tar.bz2 && \
    cd ruby && \
    bundle install ${BUNDLE_OPTIONS} && \
    bundle exec gem uninstall --force google-protobuf grpc && \
    BUNDLE_FORCE_RUBY_PLATFORM=true bundle install ${BUNDLE_OPTIONS} && \
    cd .. && \
    cp -r ${BUILD_DIR}/ruby /srv/gitaly-ruby && \
    chown ${GITLAB_USER}:${GITLAB_USER} /srv/gitaly-ruby && \
    touch .ruby-bundle && \
    make install && \
    mkdir -p /etc/gitaly && \
    rm -rf ${BUILD_DIR} /srv/gitaly-ruby/spec  /srv/gitaly-ruby/features && \
    /build-scripts/cleanup-gems /usr/lib/ruby/gems

RUN mkdir -p /var/log/gitaly && \
    touch /var/log/gitaly/gitaly.log && \
    touch /var/log/gitaly/gitlab-shell.log && chown -R ${GITLAB_USER} /var/log/gitaly

# Add scripts
COPY scripts/ /scripts/
COPY config.toml /etc/gitaly/config.toml
COPY gitconfig /usr/local/etc/gitconfig

RUN chown -R ${GITLAB_USER}:${GITLAB_USER} /etc/gitaly /scripts
USER ${GITLAB_USER}:${GITLAB_USER}

ENV CONFIG_TEMPLATE_DIRECTORY=/etc/gitaly

CMD "/scripts/process-wrapper"

VOLUME /var/log/gitaly

HEALTHCHECK --interval=30s --timeout=10s --retries=5 \
CMD /scripts/healthcheck
