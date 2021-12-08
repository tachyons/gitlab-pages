#!/bin/bash

set -xe

export HOME=/home/${GITLAB_USER}
export USER=${GITLAB_USER}
export USERNAME=${GITLAB_USER}

/scripts/set-config "${CONFIG_TEMPLATE_DIRECTORY}" "${CONFIG_DIRECTORY:=$CONFIG_TEMPLATE_DIRECTORY}"

exec /scripts/exec-env "$@"
