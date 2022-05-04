# ARGS for entire build
ARG KUBECTL_VERSION="1.18.20"
ARG FROM_IMAGE="debian"
FROM ${FROM_IMAGE}:bullseye-slim

# ARG must be redeclared after each FROM
ARG KUBECTL_VERSION
ARG YQ_VERSION="4.24.5"

# Providing to environment, for easy information gathering.
ENV KUBECTL_VERSION ${KUBECTL_VERSION}

RUN apt-get update \
  && apt-get install -y --no-install-recommends curl ca-certificates libssl1.1 openssh-client \
  && curl --retry 6 -LsfO https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl \
  && chmod +x kubectl \
  && mv kubectl /usr/local/bin/kubectl \
  && curl --retry 6 -LsfO https://github.com/mikefarah/yq/releases/download/v${YQ_VERSION}/yq_linux_amd64 \
  && chmod +x yq_linux_amd64 \
  && mv yq_linux_amd64 /usr/local/bin/yq

# Default to non-root user
USER 65534:65534
# kubectl needs a writable HOME (gitlab-org/charts/gitlab#3021)
ENV HOME=/tmp/kube
