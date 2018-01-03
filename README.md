## GitLab Pages Daemon

[![build status](https://gitlab.com/gitlab-org/gitlab-pages/badges/master/build.svg)](https://gitlab.com/gitlab-org/gitlab-pages/commits/master)

[![coverage report](https://gitlab.com/gitlab-org/gitlab-pages/badges/master/coverage.svg)](https://gitlab.com/gitlab-org/gitlab-pages/commits/master)

This is a simple HTTP server written in Go, made to serve GitLab Pages with
CNAMEs and SNI using HTTP/HTTP2. The minimum supported Go version is 1.8.

This is made to work in small to medium-scale environments. Start-up time scales
with the number of projects being served, so the daemon is currently unsuitable
for very large-scale environments.

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

1. When client initiates the TLS connection, the GitLab-Pages daemon looks in
  the generated configuration for virtual hosts. If present, it uses the TLS
  key and certificate in `config.json`, otherwise it falls back to the global
  configuration.

2. When client connects to a HTTP port the GitLab-Pages daemon looks in the
   generated configuration for a matching virtual host.

3. The URL.Path is split into `/<project>/<subpath>` and the daemon tries to
   load: `pages-root/group/project/public/subpath`.

4. If the file is not found, it will try to load `pages-root/group/<host>/public/<URL.Path>`.

5. If requested path is a directory, the `index.html` will be served.

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
$ ./gitlab-pages -listen-https "" -listen-http ":8090" -pages-root path/to/gitlab/shared/pages -pages-domain example.com
```

### Run daemon **in secure mode**

When compiled with `CGO_ENABLED=0` (which is the default), `gitlab-pages` is a
static binary and so can be run in chroot with dropped privileges.

To enter this mode, run `gitlab-pages` as the root user and pass it the
`-daemon-uid` and `-daemon-gid` arguments to specify the user you want it to run
as.

The daemon starts listening on ports and reads certificates as root, then
re-executes itself as the specified user. When re-executing it copies its own
binary to `pages-root` and changes root to that directory.

This make it possible to listen on privileged ports and makes it harder for the
process to read files outside of `pages-root`.

Example:
```
$ make
$ sudo ./gitlab-pages -listen-http ":80" -pages-root path/to/gitlab/shared/pages -pages-domain example.com -daemon-uid 1000 -daemon-gid 1000
```

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

### Enable Prometheus Metrics

For monitoring purposes, you can pass the `-metrics-address` flag when starting.
This will expose general metrics about the Go runtime and pages application for
[Prometheus](https://prometheus.io/) to scrape.

Example:
```
$ make
$ ./gitlab-pages -listen-http ":8090" -metrics-address ":9235" -pages-root path/to/gitlab/shared/pages -pages-domain example.com
```

### Cross-origin requests

GitLab Pages defaults to allowing cross-origin requests for any resource it
serves. This can be disabled globally by passing `-disable-cross-origin-requests`
when starting the daemon.

Having cross-origin requests enabled allows third-party websites to make use of
files stored on the Pages server, which allows various third-party integrations
to work. However, if it's running on a private network, this may allow websites
on the public Internet to access its contents *via* your user's browsers -
assuming they know the URL beforehand.

### Configuration

The daemon can be configured with any combination of these methods:
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
use-http2=false
```

### License

MIT
