<!-- Title suggestion: [feature flag name] Enable description of feature -->

## Summary

This issue is to rollout [the feature](ISSUE LINK) on production,
that is currently behind the `<feature-flag-name>` feature flag.

<!-- Short description of what the feature is about and link to relevant other issues. -->

## Expectations

### What are we expecting to happen?

<!-- Describe the expected outcome when rolling out this feature -->

### What might happen if this goes wrong?

<!-- Should the feature flag be turned off? Any MRs that need to be rolled back? Communication that needs to happen? What are some things you can think of that could go wrong - data loss or broken pages? -->

## Rollout Steps

- [ ] Temporarily enable with environment variable: [Link to the MR](https://gitlab.com) <!-- similar to https://gitlab.com/gitlab-com/gl-infra/k8s-workloads/gitlab-com/-/merge_requests/1500 -->
- [ ] Enable by default (optional): [Link to the MR](https://gitlab.com) <!-- similar to https://gitlab.com/gitlab-org/gitlab-pages/-/merge_requests/807 -->
- [ ] Remove the feature flag: [Link to the MR](https://gitlab.com) <!-- similar to https://gitlab.com/gitlab-org/gitlab-pages/-/merge_requests/694 -->

## Rollback Considerations/Events

<!-- List all the important considerations if the feature flag is rollback or if the feature is removed -->
- [ ] Disabling with environment variable: [Link to the MR](https://gitlab.com)
- [ ] Remove the feature flag: [Link to the MR](https://gitlab.com)

<!-- Required Labels: Do not remove -->
/label ~"feature flag" ~"type::feature" ~"feature::addition" ~backend ~"Category:Pages" ~"section::dev" ~"devops::create" ~"group::editor"
