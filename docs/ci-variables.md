# CI Variables

The GitLab Cloud Native Images [CI pipelines](pipelines.md) use variables provided by the CI environment to change build behavior between mirrors and
keep sensitive data out of the repositories.

Check the table below for more information about the various CI variables used in the pipelines.

## Build variables

| Environment Variable                          | Description |
| --------------------------------------------- | ----------- |
| GITLAB_NAMESPACE                              | Group name that containers the GitLab repos to use to pull source code from. |
| CE_PROJECT                                    | Project name for the GitLab CE rails project to use as the gitlab-rails-ce source. |
| EE_PROJECT                                    | Project name for the GitLab EE rails project to use as the gitlab-rails-ee source. |
| FETCH_DEV_ARTIFACTS_PAT                       | Access token with access to pull source assets from private locations. |
| ASSETS_IMAGE_REGISTRY_PREFIX                  | Docker registry location to pull the pre-build GitLab assets image from. |
| FORCE_IMAGE_BUILDS                            | Set to `true` to build images even when the container version matches an existing image. |
| COMPILE_ASSETS                                | Set to `true` to compile rails assets instead of using the assets image. |
| DISABLE_DOCKER_BUILD_CACHE                    | Set to any value to ensure builds run without docker build cache. |
| UBI_PIPELINE                                  | Set to any value to indicate a UBI only pipeline. |
| CE_PIPELINE                                   | Set to any value to indicate a CE only pipeline. |
| EE_PIPELINE                                   | Set to any value to indicate an EE only pipeline. |
| CUSTOM_PIPELINE                               | Set to any value to indicate an custom pipeline (don't run CE or EE specific jobs.) |
| DEPENDENCY_PROXY                              | Sets the dockerhub registry location. See [details](build.md#dependency-proxy). |

## Test variables

| Environment Variable                          | Description |
| --------------------------------------------- | ----------- |
| DANGER_GITLAB_API_TOKEN                       | GitLab API token for dangerbot to post comments to MRs. |
| DEPS_GITLAB_TOKEN                             | Token used by [dependencies.io](https://docs.dependencies.io/gitlab-ci/) to create MRs. |
| DEPS_TOKEN                                    | Token used by CI to auth to [dependencies.io](https://docs.dependencies.io/gitlab-ci/). |
| NIGHTLY                                       | Set to `true` when running a nightly build. (Busts cache). |

## Release variable

| Environment Variable                          | Description |
| --------------------------------------------- | ----------- |
| GPG_KEY_AWS_ACCESS_KEY_ID                     | Account ID to read the gpg private asset signing key (for ubi assets) from a secure s3 bucket. |
| GPG_KEY_AWS_SECRET_ACCESS_KEY                 | Account secret to read the gpg private asset signing key (for ubi assets) from a secure s3 bucket. |
| GPG_KEY_LOCATION                              | Full URI location for S3 copy command for the gpg private asset signing key. |
| GPG_KEY_PASSPHRASE                            | Passphrase for using the gpg asset signing key. |
| UBI_ASSETS_AWS_ACCESS_KEY_ID                  | Account ID to read/write from the s3 bucket containing the ubi release assets. |
| UBI_ASSETS_AWS_SECRET_ACCESS_KEY              | Account secret to read/write from the s3 bucket containing the ubi release assets. |
| UBI_ASSETS_AWS_BUCKET                         | S3 bucket name for the the ubi release assets. |
| RELEASE_API                                   | GitLab Api location for pushing release assets to. |
| UBI_RELEASE_PAT                               | GitLab Private Access Token for creating a new release object on release. |
| COM_REGISTRY                                  | Docker location of the public registry. |
| COM_CNG_PROJECT                               | Project name for the public CNG project. |
| COM_REGISTRY_PASSWORD                         | Access token for syncing to the public registry. |
| REDHAT_SECRETS_JSON                           | JSON hash of OSPID and push secrets for the RedHat images. See [build details](build.md#context). |
| SCANNING_TRIGGER_PIPELINE                     | GitLab pipeline location to trigger security scanning. |
| SCANNING_TRIGGER_TOKEN                        | Trigger Token for the security scanning project. |

## Unknown/outdated variables

| Environment Variable                          | Description |
| --------------------------------------------- | ----------- |
| BUILD_TRIGGER_TOKEN                           | |
| CHART_BUILD_TOKEN                             | |
| GCR_AUTH_CONFIG                               | |
| HELM_RELEASE_BOT_PRIVATE_KEY                  | |
| FETCH_GEMS_BUNDLE_IMAGE                       | |