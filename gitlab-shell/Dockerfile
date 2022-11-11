ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG TAG="master"

ARG GO_TAG="${TAG}"
ARG GO_IMAGE="$CI_REGISTRY_IMAGE/gitlab-go:${GO_TAG}"
ARG GITLAB_LOGGER_IMAGE="$CI_REGISTRY_IMAGE/gitlab-logger:master"

ARG GITLAB_BASE_IMAGE="${CI_REGISTRY_IMAGE}/gitlab-base:${TAG}"

ARG GOMPLATE_TAG=
ARG GOMPLATE_IMAGE="${CI_REGISTRY_IMAGE}/gitlab-gomplate:${GOMPLATE_TAG}"

FROM ${GITLAB_LOGGER_IMAGE} as gitlab-logger
FROM ${GOMPLATE_IMAGE} as gomplate

FROM ${GO_IMAGE} as build

ARG GITLAB_SHELL_VERSION=main
ARG GITLAB_NAMESPACE="gitlab-org"
ARG FETCH_ARTIFACTS_PAT
ARG CI_API_V4_URL

# install build deps
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
      gcc \
      git \
      make \
      xtail \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /assets

# Download and compile GitLab Shell
ARG CACHE_BUSTER=false
RUN export PREFIX=/assets/srv/gitlab-shell && \
    mkdir /source && \
    cd /source && \
    echo "Downloading source code from ${CI_API_V4_URL}/projects/${GITLAB_NAMESPACE}%2Fgitlab-shell/repository/archive.tar.bz2?sha=${GITLAB_SHELL_VERSION}" && \
    curl -f --retry 6 --header "PRIVATE-TOKEN: ${FETCH_ARTIFACTS_PAT}" -o gitlab-shell.tar.bz2 "${CI_API_V4_URL}/projects/${GITLAB_NAMESPACE}%2Fgitlab-shell/repository/archive.tar.bz2?sha=${GITLAB_SHELL_VERSION}" && \
    tar -xjf gitlab-shell.tar.bz2 --strip-components=1 && \
    rm gitlab-shell.tar.bz2 && \
    make build && \
    rm -rf go go_build internal/testhelper/testdata && \
    mkdir -p ${PREFIX}/bin && \
    install -m755 bin/check ${PREFIX}/bin/check && \
    install -m755 bin/gitlab-shell ${PREFIX}/bin/gitlab-shell && \
    install -m755 bin/gitlab-shell-authorized-keys-check ${PREFIX}/bin/gitlab-shell-authorized-keys-check && \
    install -m755 bin/gitlab-shell-authorized-principals-check ${PREFIX}/bin/gitlab-shell-authorized-principals-check && \
    install -m755 bin/gitlab-sshd ${PREFIX}/bin/gitlab-sshd && \
    cp LICENSE VERSION ${PREFIX}/

# Setup working directories and logs
RUN install -d /assets/srv/sshd && \
    install -d /assets/etc/ssh && \
    install -d /assets/var/log/gitlab-shell && \
    touch /assets/var/log/gitlab-shell/gitlab-shell.log

# COPY in external items
COPY --from=gitlab-logger /gitlab-logger /assets/usr/local/bin/gitlab-logger
COPY --from=gomplate /gomplate /assets/usr/local/bin/gomplate
COPY sshd_config /assets/etc/ssh/

# Set final chmods
RUN chmod -R g=u /assets

## FINAL IMAGE ##

FROM ${GITLAB_BASE_IMAGE} as final

ARG GITLAB_USER=git

# create gitlab user
# openssh daemon does not allow locked user to login, change ! to *
RUN adduser --disabled-password --gecos 'GitLab' ${GITLAB_USER} && \
    sed -i "s/${GITLAB_USER}:!/${GITLAB_USER}:*/" /etc/shadow

# install runtime deps, remove files overwritten in runtime
RUN apt-get update \
    && apt-get install -y --no-install-recommends openssh-server \
    && rm -Rf /var/lib/apt/lists/* \
    && rm -Rf /var/cache/debconf/templates.dat* \
    && rm -Rf /etc/ssh/ssh_host_* /etc/ssh/sshd_config.*

COPY --from=build --chown=${GITLAB_USER}:${GITLAB_USER} /assets/ /
# /scripts should be owned by root
COPY scripts/ /scripts
# AuthorizedKeysCommand must be owned by root, and have all parent paths owned as root
RUN mv /scripts/authorized_keys /authorized_keys && chmod 0755 /authorized_keys

USER $GITLAB_USER:$GITLAB_USER

ENV CONFIG_TEMPLATE_DIRECTORY=/srv/gitlab-shell

CMD ["/scripts/process-wrapper"]

VOLUME /var/log/gitlab-shell

HEALTHCHECK --interval=10s --timeout=3s --retries=3 \
CMD /scripts/healthcheck
