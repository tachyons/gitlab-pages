<!--
# README first!
This MR should be created on `https://gitlab.com/gitlab-org/security/gitlab-pages`.

See [the general developer security release guidelines](https://gitlab.com/gitlab-org/release/docs/blob/master/general/security/developer.md).

This merge request _must not_ close the corresponding security issue!

When submitting a merge request for gitlab-pages, CE and EE merge requests for updating pages version are both required!

-->
## Related issues

<!-- Mention the issue(s) this MR is related to -->

## Developer checklist

- [ ] Link to the original confidential issue on https://gitlab.com/gitlab-org/gitlab-pages. **Warning don't associate this MR with the security implementation issue on GitLab Security**
- [ ] MR targets `master`, or `X-Y-stable` for backports
- [ ] Milestone is set for the version this MR applies to
- [ ] Title of this MR is the same as for all backports
- [ ] A [CHANGELOG entry] has been included, with `Changelog` trailer set to `security`.
- [ ] Add a link to this MR in the `links` section of related issue
- [ ] Create a merge request in [GitLab Security](https://gitlab.com/gitlab-org/security/gitlab) bumping GitLab pages version: MR_LINK_HERE
- [ ] Assign to a Pages maintainer for review and merge

## Reviewer checklist

- [ ] Correct milestone is applied and the title is matching across all backports.
- [ ] Approve the MR. Do not merge it, release managers will assist with merging at the time of release.

[CHANGELOG entry]: https://docs.gitlab.com/ee/development/changelog.html#overview

/label ~security ~backend ~"Category:Pages"
