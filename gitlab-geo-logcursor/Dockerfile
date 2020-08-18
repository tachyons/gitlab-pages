ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/gitlab-rails-ee"
ARG TAG="master"

FROM ${FROM_IMAGE}:${TAG}

ARG DATADIR=/var/opt/gitlab
ARG CONFIG=/srv/gitlab/config
ARG GITLAB_USER=git

# Control the target for `wait-for-deps` schema check
ENV DB_SCHEMA_TARGET=geo

USER $GITLAB_USER:$GITLAB_USER

COPY scripts/ /scripts

CMD /scripts/process-wrapper

HEALTHCHECK --interval=30s --timeout=30s --retries=5 \
CMD /scripts/healthcheck
