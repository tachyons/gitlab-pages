require "gitlab-dangerfiles"

Gitlab::Dangerfiles.for_project(self) do |dangerfiles|
  dangerfiles.import_plugins
  # TODO: find a way to re-enalbe changelog https://gitlab.com/gitlab-org/gitlab-pages/-/issues/736
  dangerfiles.import_dangerfiles(except: %w[changelog])
end
