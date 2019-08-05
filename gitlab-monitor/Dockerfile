ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG TAG=latest
ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/gitlab-ruby"
ARG GITLAB_MONITOR_VERSION=4.2.0
FROM ${FROM_IMAGE}:${TAG}

# https://docs.docker.com/engine/reference/builder/#understand-how-arg-and-from-interact
ARG GITLAB_USER=git
ARG GITLAB_MONITOR_VERSION

# create gitlab user
# openssh daemon does not allow locked user to login, change ! to *
RUN adduser --disabled-password --gecos 'GitLab' ${GITLAB_USER} && \
    sed -i "s/${GITLAB_USER}:!/${GITLAB_USER}:*/" /etc/shadow

# install build deps
RUN buildDeps='build-essential' && \
    apt-get update && \
    apt-get install -y --no-install-recommends $buildDeps libpq-dev && \
    rm -rf /var/lib/apt/lists/* && \
    gem install gitlab-monitor -v ${GITLAB_MONITOR_VERSION} && \
    SUDO_FORCE_REMOVE=yes apt-get purge -y --auto-remove $buildDeps

ENV CONFIG_TEMPLATE_DIRECTORY=/var/opt/gitlab-monitor/templates
ENV CONFIG_DIRECTORY=/etc/gitlab-monitor
ENV CONFIG_FILENAME=gitlab-monitor.yml

RUN mkdir -p ${CONFIG_DIRECTORY}; chown -R $GITLAB_USER ${CONFIG_DIRECTORY}

USER $GITLAB_USER:$GITLAB_USER

CMD /usr/bin/gitlab-mon web -c ${CONFIG_DIRECTORY}/${CONFIG_FILENAME}
