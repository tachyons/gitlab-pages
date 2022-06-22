## GitLab Pages

[![build status](https://gitlab.com/gitlab-org/gitlab-pages/badges/master/pipeline.svg)](https://gitlab.com/gitlab-org/gitlab-pages/commits/master)
[![coverage report](https://gitlab.com/gitlab-org/gitlab-pages/badges/master/coverage.svg)](https://gitlab.com/gitlab-org/gitlab-pages/commits/master)

This is a simple HTTP server written in Go, made to serve GitLab Pages with
CNAMEs and SNI using HTTP/HTTP2. The minimum supported Go version is v1.17.6.

### How it generates routes

1. It reads the `pages-root` directory to list all groups.
2. It looks for `config.json` files in `pages-root/group/project` directories,
   reads them and creates mapping for custom domains and certificates.
3. It generates virtual hosts from these data.
4. Periodically (every second) it checks the `pages-root/.update` file and reads
   its content to verify if there was an update.

To reload the configuration, fill the `pages-root/.update` file with random
content. The reload will be done asynchronously, and it will not interrupt the
current requests.

### How it serves content

1. When a client initiates the TLS connection, GitLab Pages looks in
  the generated configuration for virtual hosts. If present, it uses the TLS
  key and certificate in `config.json`, otherwise it falls back to the global
  configuration.
1. When a client connects to an HTTP port, GitLab Pages looks in the
   generated configuration for a matching virtual host.
1. The URL.Path is split into `/<project>/<subpath>` and Pages tries to
   load: `pages-root/group/project/public/subpath`.
1. If the file is not found, it will try to load `pages-root/group/<host>/public/<URL.Path>`.
1. If requested path is a directory, the `index.html` file is served.
6. If `.../path.gz` exists, it will be served instead of the main file, with
   a `Content-Encoding: gzip` header. This allows compressed versions of the
   files to be precalculated, saving CPU time and network bandwidth.

### HTTPS only domains

Users have the option to enable "HTTPS only pages" on a per-project basis.
This option is also enabled by default for all newly-created projects.

When the option is enabled, a project's `config.json` will contain an
`https_only` attribute.

When the `https_only` attribute is found in the root context, any project pages
served over HTTP via the group domain (i.e. `username.gitlab.io`) will be 301
redirected to HTTPS.

When the attribute is found in a custom domain's configuration, any HTTP
requests to this domain will likewise be redirected.

If the attribute's value is false, or the attribute is missing, then
the content will be served to the client over HTTP.

### How it should be run?

Ideally the GitLab Pages should run without any load balancer in front of it.

If a load balancer is required, the HTTP can be served in HTTP mode. For HTTPS
traffic, the load balancer should be run in TCP mode. If the load balancer is
run in SSL-offloading mode, custom TLS certificates will not work.

### How to run it

Example:
```
$ make
$ ./gitlab-pages -listen-http ":8090" -pages-root path/to/gitlab/shared/pages -pages-domain example.com
```

To run on HTTPS ensure you have a root certificate key pair available

```
$ make
$ ./gitlab-pages -listen-https ":9090" -root-cert=path/to/example.com.crt -root-key=path/to/example.com.key -pages-root path/to/gitlab/shared/pages -pages-domain example.com
```

### Getting started with development

See the [contributing documentation](https://docs.gitlab.com/ee/development/pages/)

### Listen on multiple ports

Each of the `listen-http`, `listen-https` and `listen-proxy` arguments can be
provided multiple times. Gitlab Pages will accept connections to them all.

Example:
```
$ make
$ ./gitlab-pages -listen-http "10.0.0.1:8080" -listen-https "[fd00::1]:8080" -pages-root path/to/gitlab/shared/pages -pages-domain example.com
```

This is most useful in dual-stack environments (IPv4+IPv6) where both Gitlab
Pages and another HTTP server have to co-exist on the same server.


#### Listening behind a reverse proxy

When `listen-proxy` is used please make sure that your reverse proxy solution is configured to strip the [RFC7239 Forwarded headers](https://tools.ietf.org/html/rfc7239).

We use `gorilla/handlers.ProxyHeaders` middleware. For more information please review the [gorilla/handlers#ProxyHeaders](https://godoc.org/github.com/gorilla/handlers#ProxyHeaders) documentation.

> NOTE: This middleware should only be used when behind a reverse proxy like nginx, HAProxy or Apache. Reverse proxies that don't (or are configured not to) strip these headers from client requests, or where these headers are accepted "as is" from a remote client (e.g. when Go is not behind a proxy), can manifest as a vulnerability if your application uses these headers for validating the 'trustworthiness' of a request.

### PROXY protocol for HTTPS

The above `listen-proxy` option only works for plaintext HTTP, where the reverse
proxy was already able to parse the incoming HTTP traffic and inject a header for
the remote client IP.

This does not work for HTTPS which is generally proxied at the TCP level. In
order to propagate the remote client IP in this case, you can use the
[PROXY protocol](https://www.haproxy.org/download/1.8/doc/proxy-protocol.txt).
This is supported by HAProxy and some third party services such as Cloudflare.

To configure PROXY protocol support, run `gitlab-pages` with the
`listen-https-proxyv2` flag.

If you are using HAProxy as your TCP load balancer, you can configure the backend
with the `send-proxy-v2` option, like so:

```
frontend fe
    bind 127.0.0.1:12340
    mode tcp
    default_backend be

backend be
    mode tcp
    server app1 127.0.0.1:1234 send-proxy-v2
```

### GitLab access control

GitLab access control is configured with properties `auth-client-id`, `auth-client-secret`, `auth-redirect-uri`, `auth-server` and `auth-secret`. Client ID, secret and redirect uri are configured in the GitLab and should match. `auth-server` points to a GitLab instance used for authentication. `auth-redirect-uri` should be `http(s)://pages-domain/auth`. Note that if the pages-domain is not handled by GitLab pages, then the `auth-redirect-uri` should use some reserved namespace prefix (such as `http(s)://projects.pages-domain/auth`). Using HTTPS is _strongly_ encouraged. `auth-secret` is used to encrypt the session cookie, and it should be strong enough.

Example:
```
$ make
$ ./gitlab-pages -listen-http "10.0.0.1:8080" -listen-https "[fd00::1]:8080" -pages-root path/to/gitlab/shared/pages -pages-domain example.com -auth-client-id <id> -auth-client-secret <secret> -auth-redirect-uri https://projects.example.com/auth -auth-secret something-very-secret -auth-server https://gitlab.com
```

#### How it works

1. GitLab pages looks for `access_control` and `id` fields in `config.json` files
   in `pages-root/group/project` directories.
2. For projects that have `access_control` set to `true` pages will require user to authenticate.
3. When user accesses a project that requires authentication, user will be redirected
   to GitLab to log in and grant access for GitLab pages.
4. When user grants access to GitLab pages, pages will use the OAuth2 `code` to get an access
   token which is stored in the user session cookie.
5. Pages will now check user's access to a project with a access token stored in the user
   session cookie. This is done via a request to GitLab API with the user's access token.
6. If token is invalidated, user will be redirected again to GitLab to authorize pages again.

### Enable Prometheus Metrics

For monitoring purposes, you can pass the `-metrics-address` flag when starting.
This will expose general metrics about the Go runtime and pages application for
[Prometheus](https://prometheus.io/) to scrape.

Example:
```
$ make
$ ./gitlab-pages -listen-http ":8090" -metrics-address ":9235" -pages-root path/to/gitlab/shared/pages -pages-domain example.com
```

Passing the `-metrics-certificate` and `-metrics-key` flags along with `-metrics-address` flag would add TLS to the metrics.

### Structured logging

You can use the `-log-format json` option to make GitLab Pages output
JSON-structured logs. This makes it easer to parse and search logs
with tools such as [ELK](https://www.elastic.co/elk-stack).

### Cross-origin requests

GitLab Pages defaults to allowing cross-origin requests for any resource it
serves. This can be disabled globally by passing `-disable-cross-origin-requests`.

Having cross-origin requests enabled allows third-party websites to make use of
files stored on the Pages server, which allows various third-party integrations
to work. However, if it's running on a private network, this may allow websites
on the public Internet to access its contents *via* your user's browsers -
assuming they know the URL beforehand.

### SSL/TLS versions

GitLab Pages defaults to TLS 1.2 as the minimum supported TLS version. This can be
configured by using the `-tls-min-version` and `-tls-max-version` options. Accepted
values are `tls1.2`, and `tls1.3`.
See https://golang.org/src/crypto/tls/tls.go for more.

### Custom headers

To specify custom headers that should be sent with every request on GitLab pages, use the `-header` argument.

You can add as many headers as you like.

Example:
```sh
./gitlab-pages -header "Content-Security-Policy: default-src 'self' *.example.com" -header "X-Test: Testing" ...
```

### Configuration

Gitlab Pages can be configured with any combination of these methods:
1. Command-line options
1. Environment variables
1. Configuration file
1. Compile-time defaults

To see the available options and defaults, run:

```
./gitlab-pages -help
```

When using more than one method (e.g., configuration file and command-line
options), they follow the order of precedence given above.

To convert a flag name into an environment variable name:
- Drop the leading -
- Convert all - characters into _
- Uppercase the flag

e.g., `-pages-domain=example.com` becomes `PAGES_DOMAIN=example.com`

A configuration file is specified with the `-config` flag (or `CONFIG`
environment variable). Directives are specified in `key=value` format, like:

```
pages-domain=example.com
```

### License

MIT
