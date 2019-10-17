# Building Images

Building these images is done through GitLab CI. By default triggering a CI
build will use the latest upstream gitlab master, and build both gitlab-ce and
gitlab-ee versions.

For releasing new versions, the repo gets tagged by the corresponding GitLab
version by `release-tools` and the tag pipeline running on the dev.gitlab.org
mirror of the project will build the images and push them to dev registry. At
the time of publishing, these images will be copied to the com registry and
hence will be available to the public to use.

## Manually triggering the pipeline for a reference

If manual run of the pipeline needs to be done in order to build/rebuild a particular ref of GitLab,
the build needs to be triggered in this repo's GitLab project with the following variables:

|Variable|Description|Examples|
|--------|-----------|--------|
|`GITLAB_VERSION`|The GitLab ref name used to download the source code. For tags it should be the tag name.|`v11.8.0-ee`, `master`|
|`GITLAB_ASSETS_TAG`|This is used for fetching the js/css assets, and must be the slug version of the *GITLAB_VERSION*.|`v11-8-0-ee`, `11-8-stable-ee`|
|`GITLAB_TAG`|Only provided when the GitLab version is a tagged version. When set this is the tag name.|`v11.8.0-ee`|
|`GITLAB_REF_SLUG`|This is used as the docker label when the new images are built. For tags, this is the tag name. For branches, this is typically the slug version of the branch.|`v11.8.0-ee`, `my-test`|
|`GITALY_SERVER_VERSION`|The version of gitaly to build. This needs to be a tag reference that matches what is in the *GITALY_SERVER_VERSION* file in the version of GitLab being built.|`v1.12.0`|
|`GITLAB_SHELL_VERSION`|The version of gitlab-shell to build. This needs to be a tag reference that matches what is in the *GITLAB_SHELL_VERSION* file in the version of GitLab being built.|`v8.4.4`|
|`GITLAB_WORKHORSE_VERSION`|The version of workhorse to build. This needs to be a tag reference that matches what is in the *GITLAB_WORKHORSE_VERSION* file in the version of GitLab being built.|`v8.3.1`|

**For CE:**

The following variable should be present for a CE build:

- `CE_PIPELINE` - set to `true`

![ce-cng-release.png](ce-cng-release.png)

**For EE:**

The following variable should be present for a EE build:

- `EE_PIPELINE` - set to `true`

![ee-cng-release.png](ee-cng-release.png)
