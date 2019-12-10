#!/bin/bash

set -euxo pipefail

rm -f */*.tar.gz *.out failed.log

# Cleanup cached dependencies

CACHE_LOCATION=/tmp

rm ${CACHE_LOCATION}/ubi8-build-dependencies-*.tar
