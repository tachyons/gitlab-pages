ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG TAG="master"

ARG GO_TAG="${TAG}"
ARG GO_IMAGE="$CI_REGISTRY_IMAGE/gitlab-go:${GO_TAG}"
ARG GITLAB_LOGGER_IMAGE="$CI_REGISTRY_IMAGE/gitlab-logger:master"

ARG GITLAB_BASE_IMAGE="${CI_REGISTRY_IMAGE}/gitlab-base:${TAG}"

ARG GOMPLATE_TAG=
ARG GOMPLATE_IMAGE="${CI_REGISTRY_IMAGE}/gitlab-gomplate:${GOMPLATE_TAG}"

FROM ${GO_IMAGE} as builder

ARG BUILD_DIR=/tmp/build
ARG GITLAB_SHELL_VERSION=main
ARG GITLAB_NAMESPACE="gitlab-org"
ARG FETCH_ARTIFACTS_PAT
ARG CI_API_V4_URL

# install build deps
RUN buildDeps=' \
    gcc git make' \
    && apt-get update \
    && apt-get install -y --no-install-recommends $buildDeps xtail \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p ${BUILD_DIR}

# Download and compile GitLab Shell
ARG CACHE_BUSTER=false
RUN mkdir /source && \
    cd /source && \
    echo "Downloading source code from ${CI_API_V4_URL}/projects/${GITLAB_NAMESPACE}%2Fgitlab-shell/repository/archive.tar.bz2?sha=${GITLAB_SHELL_VERSION}" && \
    curl -f --retry 6 --header "PRIVATE-TOKEN: ${FETCH_ARTIFACTS_PAT}" -o gitlab-shell.tar.bz2 "${CI_API_V4_URL}/projects/${GITLAB_NAMESPACE}%2Fgitlab-shell/repository/archive.tar.bz2?sha=${GITLAB_SHELL_VERSION}" && \
    tar -xjf gitlab-shell.tar.bz2 --strip-components=1 && \
    rm gitlab-shell.tar.bz2 && \
    make build && \
    rm -rf go go_build internal/testhelper/testdata && \
    export PREFIX=/srv/gitlab-shell && \
    mkdir -p ${PREFIX}/bin && \
    install -m755 bin/check ${PREFIX}/bin/check && \
    install -m755 bin/gitlab-shell ${PREFIX}/bin/gitlab-shell && \
    install -m755 bin/gitlab-shell-authorized-keys-check ${PREFIX}/bin/gitlab-shell-authorized-keys-check && \
    install -m755 bin/gitlab-shell-authorized-principals-check ${PREFIX}/bin/gitlab-shell-authorized-principals-check && \
    install -m755 bin/gitlab-sshd ${PREFIX}/bin/gitlab-sshd && \
    cp LICENSE VERSION ${PREFIX}/

FROM ${GITLAB_LOGGER_IMAGE} as gitlab-logger
FROM ${GOMPLATE_IMAGE} as gomplate

FROM ${GITLAB_BASE_IMAGE} as final

ARG GITLAB_USER=git

# create gitlab user
# openssh daemon does not allow locked user to login, change ! to *
RUN adduser --disabled-password --gecos 'GitLab' ${GITLAB_USER} && \
    sed -i "s/${GITLAB_USER}:!/${GITLAB_USER}:*/" /etc/shadow

# install runtime deps
RUN apt-get update \
    && apt-get install -y --no-install-recommends openssh-server \
    && rm -Rf /var/lib/apt/lists/* \
    && rm -Rf /var/cache/debconf/templates.dat*

# Create a run environment for SSHD
RUN mkdir /srv/sshd && chown ${GITLAB_USER}:${GITLAB_USER} /srv/sshd

# Prepare logging directory
RUN mkdir -p /var/log/gitlab-shell && \
    touch /var/log/gitlab-shell/gitlab-shell.log && \
    chown -R ${GITLAB_USER}:${GITLAB_USER} /var/log/gitlab-shell

# Pull in final executables and configuration
COPY --from=builder /srv/gitlab-shell /srv/gitlab-shell
COPY scripts/  /scripts/
COPY sshd_config /etc/ssh/

# Pull in gitlab-logger
COPY --from=gitlab-logger /gitlab-logger /usr/local/bin/gitlab-logger

# Pull in gomplate
COPY --from=gomplate /gomplate /usr/local/bin/gomplate

# AuthorizedKeysCommand must be owned by root, and have all parent paths owned as root
RUN mv /scripts/authorized_keys /authorized_keys && chmod 0755 /authorized_keys

RUN chown -R $GITLAB_USER:$GITLAB_USER /scripts /etc/ssh && \
    install -d -o $GITLAB_USER -g $GITLAB_USER -m 755 /srv/gitlab-shell

USER $GITLAB_USER:$GITLAB_USER

ENV CONFIG_TEMPLATE_DIRECTORY=/srv/gitlab-shell

CMD ["/scripts/process-wrapper"]

VOLUME /var/log/gitlab-shell

HEALTHCHECK --interval=10s --timeout=3s --retries=3 \
CMD /scripts/healthcheck
