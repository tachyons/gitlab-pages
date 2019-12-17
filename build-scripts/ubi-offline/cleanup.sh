#!/bin/bash

set -euxo pipefail

SCRIPT_HOME="$( cd "${BASH_SOURCE[0]%/*}" > /dev/null 2>&1 && pwd )"

WORKSPACE="${SCRIPT_HOME}/build"
CACHE_LOCATION='/tmp'

rm -rf "${SCRIPT_HOME}/../.."/*/*.tar.gz "${WORKSPACE}" "${CACHE_LOCATION}"/ubi8-build-dependencies-*.tar
