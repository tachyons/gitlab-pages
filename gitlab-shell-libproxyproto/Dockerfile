ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/gitlab-shell"
ARG TAG="master"

FROM ${FROM_IMAGE}:${TAG} as shell
USER root

# From https://gitlab.com/gitlab-com/gl-infra/openssh-patches/-/releases/v0.1.0
RUN cd /tmp && \
    curl --retry 6 -o openssh-client.deb https://gitlab.com/gitlab-com/gl-infra/openssh-patches/uploads/4c4385c098a459689c4b97db0d0a14f8/openssh-client_7.9p1-10+deb10u2+gitlab+openssh79+8c55f4e4+1_amd64.deb && \
    curl --retry 6 -o openssh-server.deb https://gitlab.com/gitlab-com/gl-infra/openssh-patches/uploads/520bba5376bef780cfd0c7ccefd4338d/openssh-sftp-server_7.9p1-10+deb10u2+gitlab+openssh79+8c55f4e4+1_amd64.deb && \
    curl --retry 6 -o openssh-sftp-server.deb https://gitlab.com/gitlab-com/gl-infra/openssh-patches/uploads/520bba5376bef780cfd0c7ccefd4338d/openssh-sftp-server_7.9p1-10+deb10u2+gitlab+openssh79+8c55f4e4+1_amd64.deb && \
    dpkg -i openssh-client.deb openssh-server.deb openssh-sftp-server.deb && \
    rm openssh-client.deb openssh-server.deb openssh-sftp-server.deb
