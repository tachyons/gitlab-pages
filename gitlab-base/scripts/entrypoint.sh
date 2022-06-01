#!/bin/bash

set -e

# If in RHEL / UBI images, set ENV.
# NOTICE: ubi-micro does not have grep
if grep -q 'ID="rhel"' /etc/os-release ; then
  export HOME=/home/${GITLAB_USER}
  export USER=${GITLAB_USER}
  export USERNAME=${GITLAB_USER}
fi

/scripts/set-config "${CONFIG_TEMPLATE_DIRECTORY}" "${CONFIG_DIRECTORY:=$CONFIG_TEMPLATE_DIRECTORY}"

exec /scripts/exec-env "$@"