#!/bin/bash

set -e
/scripts/set-config "${CONFIG_TEMPLATE_DIRECTORY}" "${CONFIG_DIRECTORY:=$CONFIG_TEMPLATE_DIRECTORY}"

cd /srv/gitlab;
echo "Attempting to run '$@' as a main process";

exec "$@";
