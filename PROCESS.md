# GitLab Pages processes

## Reviewing

A contribution to GitLab Pages should generally be reviewed by at least two
people - one acting as initial reviewer, the other as a maintainer. Trivial
fixes may go straight to a maintainer. People should not merge their own
contributions.

## Versioning

GitLab Pages follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

The `X-Y-stable` branches and `master` should never have their history
rewritten. Tags should never be deleted.

## Releasing

[GitLab Pages] releases are tagged automatically by [Release Tools] when a Release Manager 
tags a GitLab version.

The version of GitLab Pages used will depend on the `GITLAB_PAGES_VERSION` file in 
the [`gitlab-org/gitlab`](https://gitlab.com/gitlab-org/gitlab) repository. This file
is managed manually, so when changes to GitLab Pages are ready to be released with GitLab, the
target commit SHA from the GitLab Pages default branch should be committed to the
`GITLAB_PAGES_VERSION` file on the `gitlab-org/gitlab` default branch. When GitLab.com
is deployed, the new version of GitLab Pages will be used. When GitLab is tagged for a monthly release,
the version of GitLab Pages from the selected deployment of GitLab will be used for tagging
GitLab Pages.

## Stable releases

Each month, when GitLab is released, a new stable branch will be created in alignment
with the version of GitLab being released. For example, release of version 15.2.0 
will result in a branch named `15-2-stable` being created on [GitLab Pages].

To backport a change: 

1. Develop an MR to fix the bug against the master branch.
1. Once ready, the MR should be merged to master, where it will be included in the next major or minor release as usual.
1. Create a merge request for `gitlab-org/gitlab` that updates `GITLAB_PAGES_VERSION` with the
merge commit SHA from the GitLab Pages default branch to deploy the changes.
1. To create a backport MR for a given stable version:
   1. Create a new branch off of the stable branch for the targeted version.
   1. Cherry-pick the commit onto the new branch.
   1. Open an MR targeting the relevant stable branch.
   1. Have the MR reviewed and merged. Note: security backports should not be merged, see [security releases](#Security releases) for more details.
1. When release managers tag a patch or security release, the stable branch will be tagged automatically.

## Security releases

This process is currently [under discussion](https://gitlab.com/gitlab-com/gl-infra/delivery/-/issues/2746). Please consult with release managers
about any process changes in the interim. 

Pages security releases are built on top of the [GitLab Security Release process]. Engineers follow
the same steps stated on the [Security Developer] guidelines with some adjustments:

- Apart from the [security merge requests] created on [GitLab Security], merge requests will also be created on [GitLab Pages Security]:
  - Merge request targeting `master` is prepared with the GitLab Pages security fix.
  - Backports are prepared for the last releases corresponding to last 3 GitLab releases.
  - Security merge requests are required to use the [merge request security template].
  - **It's important for these merge requests to not be associated with the Security Implementation Issue created on [GitLab Security], otherwise the security issue won't be considered by [Release Tools].**
- Security merge requests created on [GitLab Security] will bump the `GITLAB_PAGES_VERSION`.
- Once the merge requests on [GitLab Pages Security] are approved:
  - Maintainers of GitLab Pages will merge the security merge requests **targeting stable branches** and create a new tag for these branches.
  - Merge requests on GitLab Security are assigned to `@gitlab-release-tools-bot` so they can be automatically processed by [Release Tools].

- After the security release is published, maintainers of GitLab Pages:
  - Merge the merge requests targeting `master`.
  - Branches and tags across [GitLab Pages Security] and [GitLab Pages] are synced:
    - `Master` and stable branches.
    - Affected `v*.*.*` tags.

[GitLab Security Release process]: https://gitlab.com/gitlab-org/release/docs/blob/master/general/security/process.md
[Security Developer]: https://gitlab.com/gitlab-org/release/docs/blob/master/general/security/developer.md
[GitLab Pages Security]: https://gitlab.com/gitlab-org/security/gitlab-pages/
[security merge requests]: https://gitlab.com/gitlab-org/release/docs/blob/master/general/security/developer.md#create-merge-requests
[GitLab Security]: https://gitlab.com/gitlab-org/security/gitlab/
[merge request security template]: https://gitlab.com/gitlab-org/gitlab-pages/-/blob/master/.gitlab/merge_request_templates/Security%20Release.md
[Release Tools]: https://gitlab.com/gitlab-org/release-tools/
[GitLab Pages]: https://gitlab.com/gitlab-org/gitlab-pages
