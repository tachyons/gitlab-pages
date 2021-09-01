#!/usr/bin/env bash

set -euo pipefail

PUBLIC_PROJECT_ID='734943'     # gitlab-org/gitlab-pages
SECURITY_PROJECT_ID='15685887' # gitlab-org/security/gitlab-pages

if [[ "${SECURITY:-'0'}" == '1' ]]
then
    PROJECT_ID="$SECURITY_PROJECT_ID"
else
    PROJECT_ID="$PUBLIC_PROJECT_ID"
fi

function generate_changelog() {
    curl --header "PRIVATE-TOKEN: $TOKEN" \
        --data "version=$VERSION&branch=$BRANCH" \
        --fail \
        --silent \
        --show-error \
        "https://gitlab.com/api/v4/projects/$PROJECT_ID/repository/changelog?message=\"docs: add changelog for version $VERSION\""
}

echo 'Updating changelog on the remote branch...'

if generate_changelog
then
    echo 'Updating local branch...'
    git pull origin "$BRANCH"
    echo 'The changelog has been updated'
else
    echo "Failed to generate the changelog for version $VERSION on branch $BRANCH"
    exit 1
fi
