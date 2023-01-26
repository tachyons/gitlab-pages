require "gitlab-dangerfiles"

Gitlab::Dangerfiles.for_project(self) do |dangerfiles|
  dangerfiles.import_plugins
  # TODO: find a way to re-enalbe changelog https://gitlab.com/gitlab-org/gitlab-pages/-/issues/736
  dangerfiles.import_dangerfiles(except: %w[changelog])
end

# Identify undeployed commits only on the security mirror
SECURITY_MIRROR_PROJECT_ID = 15_685_887
if gitlab.mr_json['target_project_id'] == SECURITY_MIRROR_PROJECT_ID && gitlab.mr_json['target_branch'] == ENV['CI_DEFAULT_BRANCH']
  auto_deploy_sha = gitlab.api.file_contents('gitlab-org/gitlab', 'GITLAB_PAGES_VERSION')&.rstrip

  message("Current auto_deploy candidate version: #{auto_deploy_sha}")

  if gitlab.base_commit != auto_deploy_sha
    fail <<~MSG
      Security merge requests for `#{gitlab.mr_json['target_branch']}` must have `gitlab-org/gitlab` `GITLAB_PAGES_VERSION` content as the merge request base commit.
      Please rebase onto #{auto_deploy_sha} with `git rebase -i --onto #{auto_deploy_sha} #{gitlab.base_commit}`
      See [our documentation](https://gitlab.com/gitlab-org/release/docs/-/tree/master/components/managed-versioning/security_release.md) for details.
    MSG
  end
end
