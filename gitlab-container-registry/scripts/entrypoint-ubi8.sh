#!/bin/sh

export HOME=/home/${GITLAB_USER}
export USER=${GITLAB_USER}
export USERNAME=${GITLAB_USER}

exec "$@"