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

Pages is tightly coupled to GitLab itself. To align with GitLab's
[development month](https://gitlab.com/gitlab-org/gitlab-ce/blob/master/PROCESS.md),
new versions of GitLab Pages are released before the 7th of each month (assuming
any changes have been made). To do so:

1. For major and minor releases, create a stable branch if it doesn't already exist:

    ```shell
    git checkout -b X-Y-stable master # MAJOR.MINOR
    git push X-Y-stable https://gitlab.com/gitlab-org/gitlab-pages.git
    git push X-Y-stable https://dev.gitlab.org/gitlab/gitlab-pages.git
    ```

1. Review the list of changes since the last release and create a changelog
1. Decide on the version number by reference to the [Versioning](#versioning) section
1. [Create a new issue](https://gitlab.com/gitlab-org/gitlab-pages/issues/new) containing the changelog
1. Create a new merge request, modifying the `CHANGELOG` and `VERSION` files, targeting the correct stable branch
1. Once it's merged, create a signed+annotated tag pointing to the **merge commit** on the **stable branch**, e.g.:

    ```shell
    git fetch origin 1-0-stable
    git tag -a -s -m "Release v1.0.0" v1.0.0 origin/1-0-stable
    git push v1.0.0 https://gitlab.com/gitlab-org/gitlab-pages.git
    git push v1.0.0 https://dev.gitlab.org/gitlab/gitlab-pages.git
    ```

1. Create a merge request against [GitLab](https://gitlab.com/gitlab-org/gitlab-ce) to update `GITLAB_PAGES_VERSION`

As each release is made, the `CHANGELOG` for the stable branch will be updated
to contain content not in the **master** branch. To resolve this, the stable
branches may be merged to **master**. For more complicated merges, it may be
easier to pick just the updates to `CHANGELOG`.

## Stable releases

Typically, release tags point to a specific commit on the **master** branch. As
the Pages repository experiences a low rate of change, this allows most releases
to happen in conformance with semver, without the overhead of multiple
[stable branches](https://docs.gitlab.com/ee/workflow/gitlab_flow.html).

A bug fix may required in a particular version after the **master** branch has
moved on. This may happen between the 7th and 22nd of a release month, relating
to the **previous** release, or at any time for a security fix.

GitLab may backport security fixes for up to three releases, which may
correspond to three separate minor versions of GitLab Pages - and so three new
versions to release.

In either case, the fix should first be developed against the master branch,
taking account of the [security release workflow](https://about.gitlab.com/handbook/engineering/workflow/#security-issues)
if necessary. Once ready, the fix should be merged to master, where it will be
included in the next major or minor release as usual.

The fix may be cherry-picked into each relevant stable branch, and a new patch
release made in the same way as defined above.

When updating `GITLAB_PAGES_VERSION` in the [GitLab](https://gitlab.com/gitlab-org/gitlab-ce)
repository, you should target the relevant `X-Y-stable` branches there. In
general, these branches should only ever have the patch version of GitLab pages
incremented.
