## GitLab Pages Daemon

This is simple HTTP server written in Go made to serve GitLab Pages with CNAMEs and SNI using HTTP/HTTP2.

This is made to work in small-to-medium scale environments.
In large environment it can be time consuming to list all directories, and CNAMEs.

### How it generates routes

1. It reads the `pages-root` directory to list all groups
2. It looks for `CNAME` files in `pages-root/group/project` directory, reads them and creates mapping for custom CNAMEs.
3. It generates virtual-host from these data.
4. Periodically (every second) it checks the `pages-root` directory if it was modified to reload all mappings.

To force route refresh, CNAME reload or TLS certificate reload: `touch pages-root`.
It will be done asynchronously, not interrupting current requests. 

### How it serves content

1. When client initiates the TLS connection, the GitLab-Pages daemon looks in hash map for virtual hosts and tries to load TLS certificate from:
`pages-root/group/project/domain.{crt,key}`.

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

```
go build
./gitlab-pages -listen-https "" -listen-http ":8090" -pages-root path/to/gitlab/shared/pages
```

### License

MIT
