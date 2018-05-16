# GitLab Pages processes

## Reviewing

A contribution to GitLab Pages should generally be reviewed by at least two
people - one acting as initial reviewer, the other as a maintainer. Trivial
fixes may go straight to a maintainer. People should not merge their own
contributions.

## Versioning

GitLab Pages follows [Semantic Versioning](https://semver.org/). In its pre-1.0
state, it follows the post-1.0 rules, except that MAJOR changes are treated as
MINOR changes.

The `X-Y-stable` branches and `master` should never have their history
rewritten. Tags should never be deleted.

## Releasing

Pages is tightly coupled to GitLab itself. To align with GitLab's
[development month](https://gitlab.com/gitlab-org/gitlab-ce/blob/master/PROCESS.md),
new versions of GitLab Pages are released before the 7th of each month (assuming
any changes have been made). To do so:

* Review the list of changes since the last release and create a changelog
* Decide on the version number by reference to the [Versioning](#versioning) section
* [Create a new issue](https://gitlab.com/gitlab-org/gitlab-pages/issues/new)
  containing the changelog
* Create a new MR, modifying the `CHANGELOG` and `VERSION` files
* Once the MR is merged, create a signed+annotated tag pointing to the **merge commit** on **master**, e.g.:

    ```shell
    git tag -a -s -m "Release v1.0.0" v1.0.0
    git push v1.0.0 https://gitlab.com/gitlab-org/gitlab-pages.git
    git push v1.0.0 https://dev.gitlab.org/gitlab/gitlab-pages.git
    ```

* Create a merge request against [GitLab](https://gitlab.com/gitlab-org/gitlab-ce) to update `GITLAB_PAGES_VERSION`

## Stable releases

Typically, release tags point to a specific commit on the **master** branch. As
the Pages repository experiences a low rate of change, this allows most releases
to happen in conformance with semver, without the overhead of multiple [stable branches](https://docs.gitlab.com/ee/workflow/gitlab_flow.html).

A bug fix may required in a particular version after the **master** branch has
moved on. This may happen between the 7th and 22nd of a release month, relating
to the **previous** release, or at any time for a security fix.

GitLab may backport security fixes for up to three releases, which may
correspond to three separate minor versions of GitLab Pages - and so three new
versions to release.

In either case, the fix should first be developed against the master branch,
taking account of the [security release workflow](https://about.gitlab.com/handbook/engineering/workflow/#security-issues)
if necessary. Once ready, the fix should be merged to master.

To create the stable releases, create a stable branch if it doesn't already
exist:

```
git checkout -b X-Y-stable vX.Y.Z # MAJOR.MINOR.PATCH
git push X-Y-stable https://gitlab.com/gitlab-org/gitlab-pages.git
git push X-Y-stable https://dev.gitlab.org/gitlab/gitlab-pages.git
```

Now that the branch exists, the fix may be cherry-picked into it, and a new
release made in the same way as defined above. In these cases, the tag is
created to point at the **merge commit** of the **X-Y-stable** branch, rather
than pointing at master.

As each release is made, the `CHANGELOG` will be updated to contain content not
in the **master** branch. To resolve this, the stable branches may be merged to
**master**. For more complicated merges, it may be easier to pick just the
updates to `CHANGELOG`.

The `X-Y-stable` branch in [GitLab](https://gitlab.com/gitlab-org/gitlab-ce)
should be updated to refer to the correct stable release of GitLab Pages. In
general, these branches should only ever have their patch versions incremented.
