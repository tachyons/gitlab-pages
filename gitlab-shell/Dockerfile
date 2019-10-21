ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG TAG=latest
ARG GIT_TAG=latest

ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/gitlab-go"
ARG GIT_IMAGE=${CI_REGISTRY_IMAGE}/git-base:${GIT_TAG}

FROM ${GIT_IMAGE} AS git
FROM ${FROM_IMAGE}:${TAG} as builder

ARG BUILD_DIR=/tmp/build
ARG GITLAB_SHELL_VERSION=master
ARG GITLAB_NAMESPACE="gitlab-org"
ARG FETCH_ARTIFACTS_PAT
ARG CI_API_V4_URL

COPY --from=git /usr/local/bin/git* /usr/local/bin/
COPY --from=git /usr/local/share/git-core/ /usr/local/share/git-core/
COPY --from=git /usr/local/libexec/git-core/ /usr/local/libexec/git-core/

# install build deps
RUN buildDeps=' \
    gcc make' \
    && apt-get update \
    && apt-get install -y --no-install-recommends $buildDeps \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p ${BUILD_DIR}

# Download and compile GitLab Shell
ARG CACHE_BUSTER=false
RUN mkdir /srv/gitlab-shell && \
    cd /srv/gitlab-shell && \
    echo "Downloading source code from ${CI_API_V4_URL}/projects/${GITLAB_NAMESPACE}%2Fgitlab-shell/repository/archive.tar.bz2?sha=${GITLAB_SHELL_VERSION}" && \
    curl --header "PRIVATE-TOKEN: ${FETCH_ARTIFACTS_PAT}" -o gitlab-shell.tar.bz2 "${CI_API_V4_URL}/projects/${GITLAB_NAMESPACE}%2Fgitlab-shell/repository/archive.tar.bz2?sha=${GITLAB_SHELL_VERSION}" && \
    tar -xjf gitlab-shell.tar.bz2 --strip-components=1 && \
    rm gitlab-shell.tar.bz2 && \
    make build && \
    rm -rf go go_build

FROM ${FROM_IMAGE}:${TAG}

ARG GITLAB_USER=git

# create gitlab user
# openssh daemon does not allow locked user to login, change ! to *
RUN adduser --disabled-password --gecos 'GitLab' ${GITLAB_USER} && \
    sed -i "s/${GITLAB_USER}:!/${GITLAB_USER}:*/" /etc/shadow

# install runtime deps
RUN apt-get update \
  && apt-get install -y --no-install-recommends openssh-server \
  && rm -rf /var/lib/apt/lists/*

# Create a run environment for SSHD
RUN mkdir /srv/sshd && chown ${GITLAB_USER}:${GITLAB_USER} /srv/sshd

# Prepare logging directory
RUN mkdir -p /var/log/gitlab-shell && \
    touch /var/log/gitlab-shell/gitlab-shell.log && \
    chown -R ${GITLAB_USER}:${GITLAB_USER} /var/log/gitlab-shell

# Pull in final executables and configuration
COPY --from=builder /srv/gitlab-shell/ /srv/gitlab-shell/
COPY scripts/  /scripts/
COPY sshd_config /etc/ssh/

# AuthrorizedKeysCommand must be owned by root, and have all parent paths owned as root
RUN mv /scripts/authorized_keys /authorized_keys && chmod 0755 /authorized_keys

RUN chown -R $GITLAB_USER:$GITLAB_USER /srv/gitlab-shell /scripts /etc/ssh

USER $GITLAB_USER:$GITLAB_USER

ENV CONFIG_TEMPLATE_DIRECTORY=/srv/gitlab-shell

CMD "/scripts/process-wrapper"

VOLUME /var/log/gitlab-shell

HEALTHCHECK --interval=10s --timeout=3s --retries=3 \
CMD /scripts/healthcheck
