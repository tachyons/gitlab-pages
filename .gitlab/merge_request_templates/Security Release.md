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

- [ ] Link to the developer security workflow issue on https://gitlab.com/gitlab-org/security/gitlab
- [ ] MR targets `master`, or `X-Y-stable` for backports
- [ ] Milestone is set for the version this MR applies to
- [ ] Title of this MR is the same as for all backports
- [ ] A CHANGELOG entry is added
- [ ] Add a link to this MR in the `links` section of related issue
- [ ] Set up an CE MR: CE_MR_LINK_HERE
- [ ] Set up an EE MR: EE_MR_LINK_HERE
- [ ] Assign to a Pages maintainer for review and merge

## Reviewer checklist

- [ ] Correct milestone is applied and the title is matching across all backports
- [ ] Merge this merge request
- [ ] Create corresponding tag and push it to https://gitlab.com/gitlab-org/security/gitlab-pages

/label ~security
