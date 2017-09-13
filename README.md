## GitLab Pages Daemon

[![build status](https://gitlab.com/gitlab-org/gitlab-pages/badges/master/build.svg)](https://gitlab.com/gitlab-org/gitlab-pages/commits/master)

[![coverage report](https://gitlab.com/gitlab-org/gitlab-pages/badges/master/coverage.svg)](https://gitlab.com/gitlab-org/gitlab-pages/commits/master)

This is simple HTTP server written in Go made to serve GitLab Pages with CNAMEs and SNI using HTTP/HTTP2.
The minimum supported Go version is 1.8.

This is made to work in small-to-medium scale environments.
In large environment it can be time consuming to list all directories, and CNAMEs.

### How it generates routes

1. It reads the `pages-root` directory to list all groups
2. It looks for `config.json` file in `pages-root/group/project` directory, reads them and creates mapping for custom domains and certificates.
3. It generates virtual-host from these data.
4. Periodically (every second) it checks the `pages-root/.update` file and reads its content to verify if there was update.

To force route refresh, reload of configs fill the `pages-root/.update` with random content.
The reload will be done asynchronously, and it will not interrupt the current requests.

### How it serves content

1. When client initiates the TLS connection, the GitLab-Pages daemon looks in hash map for virtual hosts and tries to use loaded from `config.json` certificate.

2. When client asks HTTP server the GitLab-Pages daemon looks in hash map for registered virtual hosts.

3. The URL.Path is split into `/<project>/<subpath>` and we daemon tries to load: `pages-root/group/project/public/subpath`.

4. If file was not found it will try to load `pages-root/group/<host>/public/<URL.Path>`.

5. If requested path is directory, the `index.html` will be served.

### How it should be run?

Ideally the GitLab Pages should run without load balancer.

If load balancer is required, the HTTP can be served in HTTP mode.
For HTTPS traffic load balancer should be run in TCP-mode.
If load balancer is run in SSL-offloading mode the custom TLS certificate will not work.

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

The daemon start listening on ports as root, reads certificates as root and re-executes itself as specified user.
When re-executing it copies it's own binary to `pages-root` and changes root to that directory.

This make it possible to listen on privileged ports and makes it harded the process to read files outside of `pages-root`.

Example:
```
$ make
$ sudo ./gitlab-pages -listen-http ":80" -pages-root path/to/gitlab/shared/pages -pages-domain example.com -daemon-uid 1000 -daemon-gid 1000
```

### Listen on multiple ports

Each of the `listen-http`, `listen-https` and `listen-proxy` arguments can be provided multiple times. Gitlab Pages will accept connections to them all.

Example:
```
$ make
$ ./gitlab-pages -listen-http "10.0.0.1:8080" -listen-https "[fd00::1]:8080" -pages-root path/to/gitlab/shared/pages -pages-domain example.com
```

This is most useful in dual-stack environments (IPv4+IPv6) where both Gitlab Pages and another HTTP server have to co-exist on the same server.

### Enable Prometheus Metrics

For monitoring purposes, one could pass the `-metrics-address` flag when
starting. This will expose general metrics about the Go runtime and pages
application for [Prometheus](https://prometheus.io/) to scrape.

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
