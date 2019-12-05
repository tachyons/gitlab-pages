# Copyright 2017 Ismail KABOUBI
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

ARG CI_REGISTRY_IMAGE=registry.gitlab.com/gitlab-org/build/cng
ARG KUBECTL_VERSION=1.13.12

FROM ${CI_REGISTRY_IMAGE}/kubectl:${KUBECTL_VERSION} AS kubectl

FROM debian:stretch-slim

ENV REDIS_VERSION 3.2.12

RUN apt-get update \
  && apt-get install -y --no-install-recommends \
     ca-certificates \
     sed \
     curl \
     make \
     gcc \
     libc6-dev \
  && curl -fsSL "http://download.redis.io/releases/redis-${REDIS_VERSION}.tar.gz" | tar -xzC /tmp \
  && cd /tmp/redis-${REDIS_VERSION} \
  && make distclean \
  && make \
  && make install \
  && cd / \
  && apt-get -y purge gcc make libc6-dev \
  && rm -rf /tmp/redis-${REDIS_VERSION} \
  && rm -rf /var/lib/apt/lists/*

# Add kubectl to enable sentinels to change labels during promotion
COPY --from=kubectl /usr/local/bin/kubectl /usr/local/bin/kubectl

COPY assets/ /

CMD [ "touch /var/log/redis/runner.txt; chmod +w /var/log/redis/runner.txt; chmod +x /usr/local/bin/redis-launcher.sh" ]
CMD [ "/usr/local/bin/redis-launcher.sh | tee /var/log/redis/runner.txt" ]

ENTRYPOINT [ "bash", "-c" ]
