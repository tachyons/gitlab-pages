listen-proxy=0.0.0.0:8090
pages-domain=pages.127.0.0.1.nip.io
pages-root=/srv/gitlab/shared/pages
log-format=json
log-verbose
redirect-http=false
use-http2=true
artifacts-server=http://workhorse:8181/api/v4
gitlab-server=http://workhorse:8181
api-secret-key=/etc/gitlab-pages/.gitlab_pages_secret