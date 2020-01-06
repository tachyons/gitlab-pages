ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG TAG=latest
ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/gitlab-ruby"
ARG GITLAB_EXPORTER_VERSION=5.1.0
FROM ${FROM_IMAGE}:${TAG}

# https://docs.docker.com/engine/reference/builder/#understand-how-arg-and-from-interact
ARG GITLAB_USER=git
ARG GITLAB_EXPORTER_VERSION

# create gitlab user
# openssh daemon does not allow locked user to login, change ! to *
RUN adduser --disabled-password --gecos 'GitLab' ${GITLAB_USER} && \
    sed -i "s/${GITLAB_USER}:!/${GITLAB_USER}:*/" /etc/shadow

# install build deps
RUN buildDeps='build-essential' && \
    apt-get update && \
    apt-get install -y --no-install-recommends $buildDeps libpq-dev && \
    rm -rf /var/lib/apt/lists/* && \
    gem install gitlab-exporter -v ${GITLAB_EXPORTER_VERSION} && \
    SUDO_FORCE_REMOVE=yes apt-get purge -y --auto-remove $buildDeps

ENV CONFIG_TEMPLATE_DIRECTORY=/var/opt/gitlab-exporter/templates
ENV CONFIG_DIRECTORY=/etc/gitlab-exporter
ENV CONFIG_FILENAME=gitlab-exporter.yml

RUN mkdir -p ${CONFIG_DIRECTORY}; chown -R $GITLAB_USER ${CONFIG_DIRECTORY}

USER $GITLAB_USER:$GITLAB_USER

CMD /usr/bin/gitlab-exporter web -c ${CONFIG_DIRECTORY}/${CONFIG_FILENAME}
