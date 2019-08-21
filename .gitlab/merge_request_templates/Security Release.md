<!--
# README first!
This MR should be created on `dev.gitlab.org`.

See [the general developer security release guidelines](https://gitlab.com/gitlab-org/release/docs/blob/master/general/security/developer.md).

This merge request _must not_ close the corresponding security issue!

When submitting a merge request for gitlab-pages, CE and EE merge requests for updating pages version are both required!

-->
## Related issues

<!-- Mention the issue(s) this MR is related to -->

## Developer checklist

- [ ] Link to the developer security workflow issue on `dev.gitlab.org`
- [ ] MR targets `master`, or `X-Y-stable` for backports
- [ ] Milestone is set for the version this MR applies to
- [ ] Title of this MR is the same as for all backports
- [ ] A [CHANGELOG entry](https://docs.gitlab.com/ee/development/changelog.html) is added without a `merge_request` value, with `type` set to `security`
- [ ] Add a link to this MR in the `links` section of related issue
- [ ] Set up an CE MR: CE_MR_LINK_HERE
- [ ] Set up an EE MR: EE_MR_LINK_HERE
- [ ] Assign to a Pages maintainer for review and merge

## Reviewer checklist

- [ ] Correct milestone is applied and the title is matching across all backports
- [ ] Merge this merge request
- [ ] Create corresponding tag and push it to `dev.gitlab.org`

/label ~security
