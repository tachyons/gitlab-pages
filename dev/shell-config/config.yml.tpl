{%- $daemon := env.Getenv "SSH_DAEMON" -%}
# This file is managed by gitlab-ctl. Manual changes will be
# erased! To change the contents below, edit /etc/gitlab/gitlab.rb
# and run `sudo gitlab-ctl reconfigure`.

# GitLab user. git by default
user: git

# Url to gitlab instance. Used for api calls. Should end with a slash.
gitlab_url: "http://workhorse:8181/"

secret_file: /srv/gitlab-secrets/.gitlab_shell_secret

# File used as authorized_keys for gitlab user
auth_file: "/home/git/.ssh/authorized_keys"

# Log file.
# Default is gitlab-shell.log in the root directory.
{% if eq $daemon "gitlab-sshd" %}
log_file: "/dev/stdout"
{% else %}
log_file: "/var/log/gitlab-shell/gitlab-shell.log"
{% end %}


# Log level. INFO by default
log_level: INFO

# Audit usernames.
# Set to true to see real usernames in the logs instead of key ids, which is easier to follow, but
# incurs an extra API call on every gitlab-shell command.
audit_usernames: false

{% if eq $daemon "gitlab-sshd" %}
# This section configures the built-in SSH server. Ignored when running on OpenSSH.
sshd:
+  # Address which the SSH server listens on. Defaults to [::]:2222.
  listen: "[::]:2222"
  # Address which the server listens on HTTP for monitoring/health checks. Defaults to 0.0.0.0:9122.
  web_listen: "0.0.0.0:9122"
  # Maximum number of concurrent sessions allowed on a single SSH connection. Defaults to 100.
  concurrent_sessions_limit: 100
  # SSH host key files.
  host_key_files:
  {%- range file.Walk "/etc/ssh" %}
    {%- if filepath.Match "/etc/ssh/ssh_host_*_key" . %}
    - {%.%}
    {%- end %}
  {%- end %}
{% end %}

{% if env.Getenv "CUSTOM_HOOKS_DIR" %}
# Parent directory for global custom hook directories (pre-receive.d, update.d, post-receive.d)
# Default is hooks in the gitlab-shell directory.
custom_hooks_dir: "{% env.Getenv "CUSTOM_HOOKS_DIR" %}"
{% end %}
