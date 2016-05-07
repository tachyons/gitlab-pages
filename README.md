## GitLab Pages Daemon

This is simple HTTP server written in Go made to serve GitLab Pages with CNAMEs and SNI using HTTP/HTTP2.

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
go build
./gitlab-pages -listen-https "" -listen-http ":8090" -pages-root path/to/gitlab/shared/pages -pages-domain example.com
```

### Run daemon **in secure mode**

The daemon can be run in chroot with dropped privileges.

Run daemon as root user and pass the `-daemon-uid` and `-daemon-gid`.

The daemon start listening on ports as root, reads certificates as root and re-executes itself as specified user.
When re-executing it copies it's own binary to `pages-root` and changes root to that directory.

This make it possible to listen on privileged ports and makes it harded the process to read files outside of `pages-root`.

Example:
```
go build
sudo ./gitlab-pages -listen-http ":80" -pages-root path/to/gitlab/shared/pages -pages-domain example.com -daemon-uid 1000 -daemon-gid 1000
```

### Listen on multiple ports

Each of the `listen-http`, `listen-https` and `listen-proxy` arguments can be provided multiple times. Gitlab Pages will accept connections to them all.

Example:
```
go build
./gitlab-pages -listen-http "10.0.0.1:8080" -listen-https "[fd00::1]:8080" -pages-root path/to/gitlab/shared/pages -pages-domain example.com
```

This is most useful in dual-stack environments (IPv4+IPv6) where both Gitlab Pages and another HTTP server have to co-exist on the same server.

### License

MIT
