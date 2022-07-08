shutdown_timeout = "10s"

[redis]
URL = "tcp://redis:6379"
Password = "redis-meercat"

[image_resizer]
  max_scaler_procs = 100
  max_filesize = 250000

[[listeners]]
network = "tcp"
addr = "0.0.0.0:{% env.Getenv "GITLAB_WORKHORSE_LISTEN_PORT" "8181" | conv.Atoi %}"

{%- if (env.Getenv "GITLAB_WORKHORSE_LISTEN_TLS" "false" | conv.ToBool) %}
[listeners.tls]
  certificate = "/etc/gitlab/gitlab-workhorse/tls.crt"
  key = "/etc/gitlab/gitlab-workhorse/tls.key"
{%- end %}

{%- if (env.Getenv "GITLAB_WORKHORSE_METRICS_ENABLED" "false" | conv.ToBool) %}
[metrics_listener]
network = "tcp"
addr = "0.0.0.0:{% env.Getenv "GITLAB_WORKHORSE_METRICS_LISTEN_PORT" "9229" | conv.Atoi %}"
{%-   if (env.Getenv "GITLAB_WORKHORSE_METRICS_TLS_ENABLED" "false" | conv.ToBool) %}
[metrics_listener.tls]
certificate = "/etc/gitlab/gitlab-workhorse/tls.crt"
key = "/etc/gitlab/gitlab-workhorse/tls.key"
{%-   end %}
{%- end %}