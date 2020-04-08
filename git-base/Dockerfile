ARG CI_REGISTRY_IMAGE="registry.gitlab.com/gitlab-org/build/cng"
ARG TAG=latest
ARG FROM_IMAGE="$CI_REGISTRY_IMAGE/gitlab-go"

FROM ${FROM_IMAGE}:${TAG}

ARG BUILD_DIR=/tmp/build
ARG GIT_VERSION=2.26.0

RUN buildDeps=' \
    make \
    cmake \
    gcc \
    g++ \
    libicu-dev \
    libpcre2-dev \
    libcurl4-gnutls-dev \
    pkg-config \
    sudo' \
    && apt-get update \
    && apt-get install -y --no-install-recommends $buildDeps \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p ${BUILD_DIR}

RUN cd ${BUILD_DIR} \
    && curl -so git.tar.gz "https://mirrors.edge.kernel.org/pub/software/scm/git/git-${GIT_VERSION}.tar.gz" \
    && tar -xf git.tar.gz \
    && cd ${BUILD_DIR}/git-${GIT_VERSION} \
    && echo "USE_LIBPCRE2=YesPlease \n\
NO_PERL=YesPlease \n\
NO_EXPAT=YesPlease \n\
NO_TCLTK=YesPlease \n\
NO_GETTEXT=YesPlease\n\
NO_PYTHON=YesPlease \n\
NO_INSTALL_HARDLINKS=YesPlease \n\
NO_R_TO_GCC_LINKER=YesPlease" > config.mak \
    && make all prefix=/usr/local \
    && make install prefix=/usr/local \
    && cd .. \
    && rm -rf git.tar.gz git-${GIT_VERSION}
