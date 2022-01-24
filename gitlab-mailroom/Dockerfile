ARG TAG="master"
ARG FROM_IMAGE=registry.gitlab.com/gitlab-org/build/cng/gitlab-ruby

FROM ${FROM_IMAGE}:${TAG}

ARG MAILROOM_VERSION=0.0.19
ARG GITLAB_USER=git

RUN adduser --disabled-password --gecos 'GitLab' ${GITLAB_USER} && \
    sed -i "s/${GITLAB_USER}:!/${GITLAB_USER}:*/" /etc/shadow

COPY scripts/  /scripts/
COPY shared/build-scripts/ /build-scripts

RUN MAILROOM_VERSION=${MAILROOM_VERSION} /scripts/install-dependencies && rm /scripts/install-dependencies
CMD /usr/bin/mail_room -c /var/opt/gitlab/mail_room.yml --log-exit-as json
