ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG TAG=latest
ARG SHELL_CONTAINER=latest
ARG FROM_SHELL_IMAGE="$CI_REGISTRY_IMAGE/gitlab-shell"
ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/git-base"

FROM ${FROM_SHELL_IMAGE}:${SHELL_CONTAINER} as shell
FROM ${FROM_IMAGE}:${TAG}

ARG GITLAB_USER=git
ARG GITALY_VERSION=v0.96.1
ARG BUILD_DIR=/tmp/build
ARG GITLAB_NAMESPACE="gitlab-org"
ARG FETCH_ARTIFACTS_PAT
ARG CI_API_V4_URL
ARG BUNDLE_OPTIONS="--jobs 4 --without development test"

# install runtime deps. openssh-client is required so that SSH client binaries
# are present for repository mirroring.
RUN apt-get update \
    && apt-get install -y --no-install-recommends libicu57 net-tools openssh-client

# create gitlab user
# openssh daemon does not allow locked user to login, change ! to *
RUN adduser --disabled-password --gecos 'GitLab' ${GITLAB_USER} && \
      sed -i "s/${GITLAB_USER}:!/${GITLAB_USER}:*/" /etc/shadow

# create default repository directory
RUN sudo -u ${GITLAB_USER} -H mkdir -p /home/${GITLAB_USER}/repositories

# adjust git settings
RUN sudo -u ${GITLAB_USER} -H git config --global gc.auto 0 && \
    sudo -u ${GITLAB_USER} -H git config --global core.autocrlf input && \
    sudo -u ${GITLAB_USER} -H git config --global repack.writeBitmaps true && \
    sudo -u ${GITLAB_USER} -H git config --global pack.threads 1 && \
    sudo -u ${GITLAB_USER} -H git config --global receive.fsckobjects true && \
    sudo -u ${GITLAB_USER} -H git config --global receive.advertisepushoptions true && \
    sudo -u ${GITLAB_USER} -H git config --global transfer.hiderefs ^refs/tmp/ && \
    sudo -u ${GITLAB_USER} -H git config --global --add transfer.hiderefs ^refs/keep-around/ && \
    sudo -u ${GITLAB_USER} -H git config --global --add transfer.hiderefs ^refs/remotes/

# Download and compile Gitaly
ARG CACHE_BUSTER=false
RUN cd ${BUILD_DIR} && \
    echo "Downloading source code from ${CI_API_V4_URL}/projects/${GITLAB_NAMESPACE}%2Fgitaly/repository/archive.tar.bz2?sha=${GITALY_VERSION}" && \
    curl --header "PRIVATE-TOKEN: ${FETCH_ARTIFACTS_PAT}" -o gitaly.tar.bz2 "${CI_API_V4_URL}/projects/${GITLAB_NAMESPACE}%2Fgitaly/repository/archive.tar.bz2?sha=${GITALY_VERSION}" && \
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
    rm -rf ${BUILD_DIR} /srv/gitaly-ruby/spec  /srv/gitaly-ruby/features

# Include Shell
COPY --from=shell --chown=git /srv/gitlab-shell /srv/gitlab-shell

RUN cp /srv/gitlab-shell/config.yml.example /etc/gitaly/shell-config.yml && ln -s /etc/gitaly/shell-config.yml /srv/gitlab-shell/config.yml

RUN mkdir -p /var/log/gitaly && \
    touch /var/log/gitaly/gitaly.log && \
    touch /var/log/gitaly/gitlab-shell.log && chown -R ${GITLAB_USER} /var/log/gitaly

# Add scripts
COPY scripts/  /scripts/
COPY config.toml /etc/gitaly/config.toml

RUN chown -R ${GITLAB_USER}:${GITLAB_USER} /etc/gitaly /scripts
USER ${GITLAB_USER}:${GITLAB_USER}

ENV CONFIG_TEMPLATE_DIRECTORY=/etc/gitaly

CMD "/scripts/process-wrapper"

VOLUME /var/log/gitaly

HEALTHCHECK --interval=30s --timeout=10s --retries=5 \
CMD /scripts/healthcheck
