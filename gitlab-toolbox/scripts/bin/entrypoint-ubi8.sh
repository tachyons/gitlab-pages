#!/bin/bash

set -e

export HOME=/home/${GITLAB_USER}
export USER=${GITLAB_USER}
export USERNAME=${GITLAB_USER}

/scripts/set-config "${CONFIG_TEMPLATE_DIRECTORY}" "${CONFIG_DIRECTORY:=$CONFIG_TEMPLATE_DIRECTORY}"

cd /srv/gitlab;
echo "Attempting to run '$@' as a main process";

exec "$@";
