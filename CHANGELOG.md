## 1.45.0 (2021-09-21)

### Fixed (1 change)

- [fix: handle 403 errors when artifacts loaded from GitLab artifacts browser](gitlab-org/gitlab-pages@159460852e41a77b7bb5d201e127f593681c370c) ([merge request](gitlab-org/gitlab-pages!584))

## 1.44.0 (2021-09-16)

### Fixed (1 change)

- [fix: unaligned 64-bit atomic operation](gitlab-org/gitlab-pages@3db4ac6aa075a157851006d4e23f1d7088a982f6) by @ax10336 ([merge request](gitlab-org/gitlab-pages!571))

## 1.43.0 (2021-08-26)

### Added (1 change)

- [Splat and placeholder support in _redirects](gitlab-org/gitlab-pages@5fff32dd852a27151a08dea0cb2a8fd2983d7f65) ([merge request](gitlab-org/gitlab-pages!458))

### Changed (2 changes)

- [feat: Add _redirects max rule count validation](gitlab-org/gitlab-pages@65188a6f442d2fa5f34be815564c151f3a52e8a7) by @nfriend ([merge request](gitlab-org/gitlab-pages!555))
- [build: bump go to 1.16](gitlab-org/gitlab-pages@e1c89e281f9af0e15a0d3035ce4ff1fe1ada2b79) by @feistel ([merge request](gitlab-org/gitlab-pages!547))

### Removed (2 changes)

- [refactor: remove chroot/jail logic](gitlab-org/gitlab-pages@6b7b256de67436735b948e63663a875d7043a8dd) by @feistel ([merge request](gitlab-org/gitlab-pages!536))
- [refactor: remove support for disk configuration source](gitlab-org/gitlab-pages@5e9447161081a73d15ae81cd21da683fde7c2e9b) by @feistel ([merge request](gitlab-org/gitlab-pages!541))

## 1.42.0 (2021-08-17)

### Changed (2 changes)

- [feat: add CORS header to HEAD requests](gitlab-org/gitlab-pages@52f82517edfa6c2c1a3220d6ab5cf1440faf2d17) ([merge request](gitlab-org/gitlab-pages!531))
- [Use internal-gitlab-server in auth-related tasks](gitlab-org/gitlab-pages@7a9c492b619078aed6f9c3f95cf21640afd63100) ([merge request](gitlab-org/gitlab-pages!507))

### Other (4 changes)

- [fix: do not fail to print --version](gitlab-org/gitlab-pages@5186f78c179578757a2673264155aa8b287a0efb) ([merge request](gitlab-org/gitlab-pages!539))
- [build: replace jwt-go with maintained fork](gitlab-org/gitlab-pages@436803f5975eefd697c83d6ad6e5c43360be8310) ([merge request](gitlab-org/gitlab-pages!533))
- [refactor: fail to start without listeners](gitlab-org/gitlab-pages@0dc345e1a3278ea4b922e2a7bf4952caf20bc139) ([merge request](gitlab-org/gitlab-pages!532))
- [ci: use gotestsum for running tests](gitlab-org/gitlab-pages@778c290a0eed19a4203c167915d42a223ebc7c5e) ([merge request](gitlab-org/gitlab-pages!528))

## 1.41.0 (2021-07-13)

### Added (1 change)

- [Include /etc/nsswitch.conf in chroot jail](gitlab-org/gitlab-pages@56273e40459345534203433d02682c4539507c73) ([merge request](gitlab-org/gitlab-pages!499))

### Fixed (1 change)

- [Fix path trimming in GitLab client](gitlab-org/gitlab-pages@3388ae9caea138923dd2e682aa0315eed9c0fcf5) ([merge request](gitlab-org/gitlab-pages!512))

### Changed (1 change)

- [Disable chroot and add daemon-enable-jail flag](gitlab-org/gitlab-pages@4d1dcf7933442c4b062b85fe26a2aa6cc75a078d) ([merge request](gitlab-org/gitlab-pages!513))

### Other (1 change)

- [Improve observability of HTTP connections](gitlab-org/gitlab-pages@2b1887c217e04f05c885202d73de06f44f315460) ([merge request](gitlab-org/gitlab-pages!515))

## 1.40.0 (2021-06-09)

### Changed (1 change)

- [Use API source by default](gitlab-org/gitlab-pages@9a30f40a785ffe586a79a75ebebabda5f513b76f) ([merge request](gitlab-org/gitlab-pages!491))

### Removed (2 changes)

- [Drop support for Go 1.13](gitlab-org/gitlab-pages@5abbb3dd66270f0124fc5c7fe4523f237337c890) ([merge request](gitlab-org/gitlab-pages!501))
- [Remove gitlab-server fallback to auth-server and artifacts server](gitlab-org/gitlab-pages@6f806a34631581486ca12e2fe3addd3f722b6477) ([merge request](gitlab-org/gitlab-pages!472))

### Other (1 change)

- [Sort lookup paths by prefix length](gitlab-org/gitlab-pages@b9ba78cf43a14a4ca1c4893c7115eed56ce1b115) ([merge request](gitlab-org/gitlab-pages!496))

## 1.39.0 (2021-05-14)

### Added (1 change)

- [Add flag enable-disk](gitlab-org/gitlab-pages@8ea4cb76586c0841456c5973168cd8451f6c6c0a) ([merge request](gitlab-org/gitlab-pages!475))

### Removed (1 change)

- [ Remove use-legacy-storage](gitlab-org/gitlab-pages@ed66d91bd5eba6aa267e15520a1a01f12d624235) ([merge request](gitlab-org/gitlab-pages!476))

### Other (2 changes)

- [Use config package in GitLab client](gitlab-org/gitlab-pages@890d01033ab17ea5a6208450ced595d12f4ce195) ([merge request](gitlab-org/gitlab-pages!474))
- [Remove gocovmerge](gitlab-org/gitlab-pages@8e83053055ab3f48232eb115e51aeb7532fe301c) ([merge request](gitlab-org/gitlab-pages!459))

## 1.38.0 (2021-04-15)

### Added (1 change)

- [Log connection attempt errors](gitlab-org/gitlab-pages@7298e8fd04d58844d2c0fd11051318b392c02ecd) ([merge request](gitlab-org/gitlab-pages!456))

### Fixed (1 change)

- [Allow serving zip from disk in chroot](gitlab-org/gitlab-pages@34e1518038a8164c090ccfbfc30ebc7850e62cf0) ([merge request](gitlab-org/gitlab-pages!457))

## 1.37.0 (2021-03-31)

### Added (1 change)

- [Instrument limit listener for saturation monitoring](gitlab-org/gitlab-pages@b2bebb9733332031654f203a42c9912831d49a15) ([merge request](gitlab-org/gitlab-pages!443))

### Changed (1 change)

- [Add FF_DISABLE_REFRESH_TEMPORARY_ERROR env variable](gitlab-org/gitlab-pages@da02f9667d9c0696538bc7ca85a84bf66a33942c) ([merge request](gitlab-org/gitlab-pages!435))

### Other (1 change)

- [Remove .GOPATH symlinks](gitlab-org/gitlab-pages@33723f8a85871797b6f38ab0340437a9b635f917) ([merge request](gitlab-org/gitlab-pages!453))

## 1.36.0 (2021-03-12)

### Added (3 changes)

- [Add GitLab cache config flags](gitlab-org/gitlab-pages@93c7957b8a2673c418f3e9620d99a5206a02adcc) ([merge request](gitlab-org/gitlab-pages!442))
- [Add use-legacy-storage flag](gitlab-org/gitlab-pages@258be795aa78afe2252e630508fa049a596251fc) ([merge request](gitlab-org/gitlab-pages!439))
- [fix(auth): make authentication scope for Pages configurable](gitlab-org/gitlab-pages@b41995a13969b2926ad265bcc769f473e48166cb)

### Fixed (1 change)

- [fix: use correlationID middleware](gitlab-org/gitlab-pages@ae9a8fb5304fca0a1dc0441cb991227320033bca) ([merge request](gitlab-org/gitlab-pages!438))

### Changed (3 changes)

- [Move config validations to separate file](gitlab-org/gitlab-pages@23ac0e80a47e578fd17cee491e8ad0af13e67d37) ([merge request](gitlab-org/gitlab-pages!440))
- [Add Cache to config pkg](gitlab-org/gitlab-pages@bc93c23e1b5ffd4acb99935c2a77966322112c50) ([merge request](gitlab-org/gitlab-pages!434))
- [Move configuration parsing to Config package](gitlab-org/gitlab-pages@b7e2085b76c11212ac41f80672d5c5f9b0287fee) ([merge request](gitlab-org/gitlab-pages!431))

### Other (1 change)

- [Add changelog generation script](gitlab-org/gitlab-pages@789cbeca36efcd135ec9ccb134d91d9487eeb034) ([merge request](gitlab-org/gitlab-pages!447))

## 1.35.0

- Fix for query strings being stripped !398
- Do not accept client-supplied X-Forwarded-For header for logs without proxy !415
- Include /etc/hosts in chroot jail !124
- Render 500 error if API is unavailable and domain info is unavailable !393
- Allow passing multiple values in `-header` with separator via config file !417
- Fix `auto` config source !424
- Allow to serve `zip` from a disk `/pages` !429

## 1.34.0

- Allow DELETE HTTP method

## 1.33.0

- Reject requests with unknown HTTP methods
- Encrypt OAuth code during auth flow

## 1.32.0

- Try to automatically use gitlab API as a source for domain information !402
- Fix https redirect loop for PROXYv2 protocol !405

## 1.31.0

- Support for HTTPS over PROXYv2 protocol !278
- Update LabKit library to v1.0.0 !397
- Add zip serving configuration flags !392
- Disable deprecated serverless serving and proxy !400

## 1.30.2

- Allow DELETE HTTP method

## 1.30.1

- Reject requests with unknown HTTP methods
- Encrypt OAuth code during auth flow

## 1.30.0

- Allow to refresh an existing cached archive when accessed !375

## 1.29.0

- Fix LRU cache metrics !379
- Upgrade go-mimedb to support new types including avif images !353
- Return 5xx instead of 404 if pages zip serving is unavailable !381
- Make timeouts for ZIP VFS configurable !385
- Improve httprange timeouts !382
- Fix caching for errored ZIP VFS archives !384

## 1.28.2

- Allow DELETE HTTP method

## 1.28.1

- Reject requests with unknown HTTP methods
- Encrypt OAuth code during auth flow

## 1.28.0

- Implement basic redirects via _redirects text file !367
- Add support for pre-compressed brotly files !359
- Add serving type to log !369
- Improve performance of ZIP serving !364
- Fix support for archives without directory structure !373

## 1.27.0

- Add more metrics for zip serving !363 !338

## 1.26.0

- Add the ability to serve web-sites from the zip archive stored in object storage !351

## 1.25.0

- No user-facing changes

## 1.24.0

- Unshare mount namespaces when creating jail !342

## 1.23.0

- Add VFS for local disk !324
- Fully support `domain-config-source=gitlab` !332

## 1.22.0

- Serve custom 404.html file for namespace domains !263
- Poll internal status API !304 !306
- Enable `domain-config-source=disk` by default Use domain config source disk !305
- Set Content-Length when Content-Encoding is set !227

## 1.21.0

- Copy certs from SSL_CERT_DIR into chroot jail !291

## 1.20.0

- Enable continuous profiling !297

## 1.19.0

- Add file size metric for disk serving !294
- Add pprof to metrics endpoint !271

## 1.18.0

- Fix proxying artifacts with escaped characters !255
- Introduce internal-gitlab-server flag to allow using the internal network for communicating to the GitLab server !276
- Increase maximum idle connections pool size from 2 to 100 !274
- Disable passing auth-related secret parameters as command line flags !269
- Fix unused idle API connection bug !275

## 1.17.0

- Extract health check in its own middleware !247
- Increase GitLab internal API response timeout !253
- Add support for proxying GitLab serverless requests !232

## 1.16.0

- Add metrics for GitLab API calls !229
- Change the way proxy headers like `X-Forwarded-For` are handled !225

## 1.15.0

- Implement support for incremental rollout of the new API based configuration source
- Add domain configuration duration (from disk) to the exported Prometheus metrics
- Make GitLab API client timeout and JWT expiry configurable

## 1.14.0

- Rollback godirwalk to v1.10.12 due to significant performance degradation

## 1.13.0

- Implement API based configuration source (not yet used)
- Update godirwalk to v1.14.0

## 1.12.0

- Add minimal support for the api-secret-key config flag (not yet used)
- Add warnings about secrets given through command-line flags
- Remove Admin gRPC api (was never used)

## 1.11.0

- Refactor domain package and extract disk serving !189
- Separate domain config source !188

## 1.10.0

- Add support for previewing artifacts that are not public !134

## 1.9.0

- Add full HTTP metrics and logging to GitLab pages using LabKit

## 1.8.1

- Limit auth cookie max-age to 10 minutes
- Use secure cookies for auth session

## 1.8.0

- Fix https downgrade in auth process
- Fix build under go-pie environment
- Change Prometheus metrics names
- Require minimum golang version 1.11 to build
- Add the ability to define custom HTTP headers for all served sites

## 1.7.2

- Fix https to http downgrade for auth process
- Limit auth cookie max-age to 10 minutes
- Use secure cookies for auth session

## 1.7.1

- Security fix for recovering gitlab access token from cookies

## 1.7.0

- Add support for Sentry error reporting

## 1.6.3

- Fix https to http downgrade for auth process
- Limit auth cookie max-age to 10 minutes
- Use secure cookies for auth session

## 1.6.2

- Security fix for recovering gitlab access token from cookies

## 1.6.1

- Fix serving acme challenges with index.html

## 1.6.0

- Use proxy from environment for http request !131
- Use STDOUT for flag outputs !132
- Prepare pages auth logs for production rollout !138
- Redirect unknown ACME challenges to the GitLab instance !141
- Disable 3DES and other insecure cipher suites !145
- Provide ability to disable old TLS versions !146

## 1.5.1

- Security fix for recovering gitlab access token from cookies

## 1.5.0

- Make extensionless URLs work !112

## 1.4.0
- Prevent wrong mimetype being set for GZipped files with unknown file extension !122
- Pages for subgroups !123
- Make content-type detection consistent between file types !126

## 1.3.1
- Fix TOCTOU race condition when serving files

## 1.3.0
- Allow the maximum connection concurrency to be set !117
- Update Prometheus vendoring to v0.9 !116
- Fix version string not showing properly !115

## 1.2.1
-  Fix 404 for project with capital letters !114

## 1.2.0
- Stop serving shadowed namespace project files !111
- Make GitLab pages support access control !94

## 1.1.0
- Fix HTTP to HTTPS redirection not working for default domains !106
- Log duplicate domain names !107
- Abort domain scan if a failure is encountered !102
- Update Prometheus vendoring !105

## 1.0.0
- Use permissive unix socket permissions !95
- Fix logic for output of domains in debug mode !98
- Add support for reverse proxy header X-Forwarded-Host !99

## 0.9.1
- Clean up the created jail directory if building the jail doesn't work !90
- Restore the old in-place chroot behaviour as a command-line option !92
- Create /dev/random and /dev/urandom when daemonizing and jailing !93

## 0.9.0
- Add gRPC admin health check !85

## 0.8.0
- Add /etc/resolv.conf and /etc/ssl/certs to pages chroot !51
- Avoid unnecessary stat calls when building domain maps !60
- Parallelize IO during the big project scan !61
- Add more logging to gitlab pages daemon !62
- Remove noisy debug logs !65
- Don't log request or referer query strings !77
- Make certificate parsing thread-safe !79

## 0.7.1
- Fix nil reference error when project is not in config.json !70

## 0.7.0
- HTTPS-only pages !50
- Switch to govendor !54
- Add logrus !55
- Structured logging !56
- Use https://github.com/jshttp/mime-db to populate the mimedb !57

## 0.6.1
- Sanitize redirects by issuing a complete URI

## 0.6.0
- Use namsral/flag to support environment vars for config !40
- Cleanup the README file !41
- Add an artifacts proxy to GitLab Pages !44 !46
- Resolve "'cannot find package' when running make" !45

## 0.5.1
- Don't serve statically-compiled `.gz` files that are symlinks

## 0.5.0
- Don't try to update domains if reading the update file fails !32
- Add CORS support to GET requests !33
- Add CONTRIBUTING.md !34
- Add basic cache directives to gitlab-pages !35
- Go 1.8 is the minimum supported version !36
- Fix HTTP/2 ALPN negotiation !37
- Add disabled-by-default status check endpoint !39

## 0.4.3
- Fix domain lookups when Pages is exposed on non-default ports

## 0.4.2
- Support for statically compressed gzip content-encoding

## 0.4.1
- Fix reading configuration for multiple custom domains

## 0.4.0
- Fix the `-redirect-http` option so it redirects from HTTP to HTTPS when enabled !21

## 0.3.2
- Only pass a metrics fd to the daemon child if a metrics address was specified

## 0.3.1
- Pass the metrics address fd to the child process

## 0.3.0
- Prometheus metrics support with `metrics-address`

## 0.2.5
- Allow listen-http, listen-https and listen-proxy to be specified multiple times

## 0.2.4
- Fix predefined 404 page content-type

## 0.2.3
- Add `-version` to command line

## 0.2.2
- Fix predefined 404 page content-type

## 0.2.1
- Serve nice GitLab branded 404 page
- Present user's error page for 404: put the 404.html in root of your pages

## 0.2.0
- Execute the unprivileged pages daemon in chroot

## 0.1.0
- Allow to run the pages daemon unprivileged (-daemon-uid, -daemon-gid)

## 0.0.0
- Initial release
