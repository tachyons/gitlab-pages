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
any changes have been made).
To do so create [release issue](https://gitlab.com/gitlab-org/gitlab-pages/issues/new?issuable_template=release) and follow the instructions.

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
versions to release. See [Security releases](#Security releases) for the details.

In either case, the fix should first be developed against the master branch.
Once ready, the fix should be merged to master, where it will be
included in the next major or minor release as usual.

The fix may be cherry-picked into each relevant stable branch, and a new patch
release made in the same way as defined above.



When updating `GITLAB_PAGES_VERSION` in the [GitLab](https://gitlab.com/gitlab-org/gitlab-ce)
repository, you should target the relevant `X-Y-stable` branches there. In
general, these branches should only ever have the patch version of GitLab pages
incremented.

## Security releases

We follow general [security release workflow](https://about.gitlab.com/handbook/engineering/workflow/#security-issues) for pages releases.
Use [Security Release](.gitlab/merge_request_templates/Security Release.md) template for security related merge requests.

### After security release has been published

Maintainer needs to manually sync tags and branches from dev.gitlab.org to gitlab.com:

- [ ] Sync `master` branch
- [ ] Sync affected `*-*-stable` branches
- [ ] Sync affected `v*.*.*` tags
