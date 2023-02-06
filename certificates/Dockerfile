## FINAL IMAGE ##
# certificates
# - Built on gitlab-base, which has ca-certificates already installed.

ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG TAG="master"
ARG GITLAB_BASE_IMAGE="${CI_REGISTRY_IMAGE}/gitlab-base:${TAG}"

FROM $GITLAB_BASE_IMAGE

COPY scripts/bundle-certificates /scripts/

VOLUME /etc/ssl/certs /usr/local/share/ca-certificates

CMD ["/scripts/bundle-certificates"]
