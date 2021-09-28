# CI Variables

The GitLab Cloud Native Images [CI pipelines](pipelines.md) use variables provided by the CI environment to change build behavior between mirrors and
keep sensitive data out of the repositories.

Check the table below for more information about the various CI variables used in the pipelines.

## Build variables

| Environment Variable                          | Description |
| --------------------------------------------- | ----------- |
| GITLAB_NAMESPACE                              | GitLab group containing the rails source code repositories named by `CE_PROJECT` and `EE_PROJECT`. |
| CE_PROJECT                                    | GitLab project containing the GitLab CE source code for the gitlab-rails-ce image. |
| EE_PROJECT                                    | GitLab project containing the GitLab EE source code for the gitlab-rails-ee image. |
| FETCH_DEV_ARTIFACTS_PAT                       | Access token with permission to pull source assets from private locations. |
| ASSETS_IMAGE_REGISTRY_PREFIX                  | Pull pre-built GitLab assets container image from specified Docker registry location. |
| COMPILE_ASSETS                                | Setting `true` generates fresh rails assets instead of copying them from the assets image.
| FORCE_IMAGE_BUILDS                            | Setting `true` builds fresh images even when existing containers match the specified version. |
| DISABLE_DOCKER_BUILD_CACHE                    | Setting any value ensures that builds run without docker build cache. |
| UBI_PIPELINE                                  | Setting to any value indicates this will be a UBI only pipeline. |
| CE_PIPELINE                                   | Setting any value indicates this will be a CE only pipeline. |
| EE_PIPELINE                                   | Setting any value indicates this will be an EE only pipeline. |
| CUSTOM_PIPELINE                               | Setting any value indicates this will be a custom pipeline (Do not run CE or EE specific jobs.) |
| DEPENDENCY_PROXY                              | Sets the dockerhub registry location. See [details](build.md#dependency-proxy). |
|GITLAB_BUNDLE_GEMFILE                          | Setting Gemfile path required by `gitlab-rails` bundle. If bundle uses the default Gemfile, just keep it unset. |

## Test variables

| Environment Variable                          | Description |
| --------------------------------------------- | ----------- |
| DANGER_GITLAB_API_TOKEN                       | GitLab API token dangerbot uses to post comments on MRs. |
| DEPS_GITLAB_TOKEN                             | Token used by [dependencies.io](https://docs.dependencies.io/gitlab-ci/) to create MRs. |
| DEPS_TOKEN                                    | Token used by CI for auth to [dependencies.io](https://docs.dependencies.io/gitlab-ci/). |
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
| RELEASE_API                                   | Target GitLab API location when pushing release assets. |
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