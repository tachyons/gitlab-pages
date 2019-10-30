#!/bin/sh

set -eu

IMPORT_RESULT=$(./bin/goimports -e -local "gitlab.com/gitlab-org/gitlab-pages" -l "$@")

if [ -n "${IMPORT_RESULT}" ]; then
  echo >&2 "Please run ./bin/goimports -w -local gitlab.com/gitlab-org/gitlab-pages -l $@"
  echo "${IMPORT_RESULT}"
  exit 1
fi
