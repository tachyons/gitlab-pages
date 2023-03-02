## 15.9.2 (2023-03-02)

No changes.

## 15.9.1 (2023-02-23)

No changes.

## 15.9.0 (2023-02-21)

No changes.

## 15.8.4 (2023-03-02)

No changes.

## 15.8.3 (2023-02-15)

No changes.

## 15.8.2 (2023-02-10)

No changes.

## 15.8.1 (2023-01-30)

No changes.

## 15.8.0 (2023-01-20)

### Security (1 change)

- [Restrict arbitrary protocol redirection to only https or http URLs](gitlab-org/gitlab-pages@973d93daeaa0e31f0cec2e09db8838cf38c67dc5) ([merge request](gitlab-org/gitlab-pages!845))

## 15.7.8 (2023-03-02)

No changes.

## 15.7.7 (2023-02-10)

No changes.

## 15.7.6 (2023-01-30)

No changes.

## 15.7.5 (2023-01-12)

No changes.

## 15.7.4 (2023-01-12)

No changes.

## 15.7.3 (2023-01-11)

No changes.

## 15.7.2 (2023-01-09)

### Security (1 change)

- [Restrict arbitrary protocol redirection to only https or http URLs](gitlab-org/security/gitlab-pages@349102115927947edd59ce0a03d1ffba3a74947f) ([merge request](gitlab-org/security/gitlab-pages!59))

## 15.7.1 (2023-01-05)

No changes.

## 15.7.0 (2022-12-21)

No changes. Same content of 1.64.0.

## 15.6.8 (2023-02-10)

No changes.

## 15.6.7 (2023-01-30)

No changes.

## 15.6.6 (2023-01-12)

No changes.

## 15.6.5 (2023-01-12)

No changes.

## 15.6.4 (2023-01-09)

### Security (1 change)

- [Restrict arbitrary protocol redirection to only https or http URLs](gitlab-org/security/gitlab-pages@c0da7401a044a31d1ccf754716880e3e4721453f) ([merge request](gitlab-org/security/gitlab-pages!58))

## 15.6.3 (2022-12-21)

No changes.

## 15.6.2 (2022-12-15)

No changes. Same content of 1.63.0.

## 15.5.9 (2023-01-12)

No changes.

## 15.5.8 (2023-01-12)

No changes.

## 15.5.7 (2023-01-09)

### Security (1 change)

- [Restrict arbitrary protocol redirection to only https or http URLs](gitlab-org/security/gitlab-pages@f14d39bbaacd76d8be26b6121e732c7327cc0d4d) ([merge request](gitlab-org/security/gitlab-pages!57))

## 15.5.6 (2022-12-15)

No changes. Same content of 1.62.0.

## 15.4.6 (2022-12-15)

No changes. Same content of 1.62.0.

## 1.64.0 (2022-12-01)

No changes.

## 1.63.0 (2022-11-10)

### Security (1 change)

- [Fix CVE-2022-32149 in golang.org/x/text](gitlab-org/gitlab-pages@7e01bfda3f59a5bcb78af4f4d3001dfa7fe1078a) ([merge request](gitlab-org/gitlab-pages!832))

### Other (1 change)

- [Add note about docs](gitlab-org/gitlab-pages@b6b2bf5a25558a1c9173b2ca55063528bc6c6c7f) ([merge request](gitlab-org/gitlab-pages!835))

## 1.62.0 (2022-07-28)

### Fixed (2 changes)

- [Fixes acme redirection issues when using a wildcard redirect](gitlab-org/gitlab-pages@a131f2bdf10e2815d79e3581fe6536b1499839b6) ([merge request](gitlab-org/gitlab-pages!819))
- [Fix data race with lookup paths](gitlab-org/gitlab-pages@9bf12d385279f8386716cc0af4372ebcd5ae6f9d) ([merge request](gitlab-org/gitlab-pages!822))

### Changed (2 changes)

- [Log ZIP archive corruption error](gitlab-org/gitlab-pages@e34ed022accdff134d7464de03069670c0e18aaf) ([merge request](gitlab-org/gitlab-pages!821))
- [Update LabKit library to v1.16.0](gitlab-org/gitlab-pages@77220ca923e02ba0c4b464421b421df0a8cd7f06) ([merge request](gitlab-org/gitlab-pages!813))

## 1.61.1 (2022-07-19)

### Security (1 change)

- [Include remote address through X-Forwarded-For in job artifact request](gitlab-org/security/gitlab-pages@3224eb6418fcc0fcb0fa5bd3203c0c0c25b4704b) ([merge request](gitlab-org/security/gitlab-pages!38))

## 1.59.0 (2022-06-13)

### Added (2 changes)

- [Add support for tls for metrics](gitlab-org/gitlab-pages@8488ef56611256c1761f93de5f8df23e07b86af4) ([merge request](gitlab-org/gitlab-pages!772))
- [Add support for socket listeners](gitlab-org/gitlab-pages@60e9cfb7e61fe9eb7141cc9e7d6e95495048bb37) by @feistel ([merge request](gitlab-org/gitlab-pages!758))

## 1.58.0 (2022-05-17)

### Changed (3 changes)

- [Use labkit for fips check](gitlab-org/gitlab-pages@21cfe26446f7862e2a65c9129ef573a1881f296d) ([merge request](gitlab-org/gitlab-pages!755))
- [Upgrade labkit to version v1.13.0](gitlab-org/gitlab-pages@6574ad0b413ee7ae5ea5b6459d8bc4523e3f492d) ([merge request](gitlab-org/gitlab-pages!739))
- [config: Default serverWriteTimeout to 0](gitlab-org/gitlab-pages@790b92165f6e29cc7bada7862cdf3f1928170b23) ([merge request](gitlab-org/gitlab-pages!741))

### Removed (3 changes)

- [Remove deprecated daemon flags](gitlab-org/gitlab-pages@de280760e40d45a05c1c7ccd99d7465ca1fba26e) ([merge request](gitlab-org/gitlab-pages!751))
- [Remove domain-config-source flag](gitlab-org/gitlab-pages@65f22020f9c1620ee567f71e1fb3a0f095225e45) by @feistel ([merge request](gitlab-org/gitlab-pages!745))
- [Remove FF_DISABLE_REFRESH_TEMPORARY_ERROR feature flag](gitlab-org/gitlab-pages@2f569aea44ddfbe443031edf9855a7ebae18095c) by @feistel ([merge request](gitlab-org/gitlab-pages!694))

### Other (1 change)

- [Replace make setup with go run and version suffixes](gitlab-org/gitlab-pages@65d0d02887b86661590e74f560cbda0d517b95c2) by @feistel ([merge request](gitlab-org/gitlab-pages!750))

## 1.57.0 (2022-04-13)

### Added (2 changes)

- [Add FIPS support](gitlab-org/gitlab-pages@2e4b84b6ac95087b96e346916b3ced662269b15d) ([merge request](gitlab-org/gitlab-pages!716))
- [add flag to parameterize zip http client timeout](gitlab-org/gitlab-pages@29cb5da295e31d18c29c544a8f43f8006b3e874d) ([merge request](gitlab-org/gitlab-pages!710))

### Fixed (1 change)

- [Increase serverWriteTimeout to avoid errors with large files](gitlab-org/gitlab-pages@f2792ec357dd90dcc4e4df929386c02929450999) ([merge request](gitlab-org/gitlab-pages!722))

### Changed (3 changes)

- [Update nonce to make it of standard size](gitlab-org/gitlab-pages@8e40856d4b14a261246b3bd8d3a2b80dd69a99e7) ([merge request](gitlab-org/gitlab-pages!719))
- [Update go-cmp](gitlab-org/gitlab-pages@8bb08f8f0b8e0366b2a3b75767d011aec35b265b) ([merge request](gitlab-org/gitlab-pages!713))

### Security (1 change)

- [fix: validate that session was issued on the same host](gitlab-org/gitlab-pages@9dbeb71c8a99ed0517b3ba44950ee63c00eb6cf6)

## 1.56.2 (2022-04-13)

### Fixed (1 change)

- [Increase serverWriteTimeout to avoid errors with large files](gitlab-org/gitlab-pages@34ca1a3c379d98a8e52e52b9ddfc81abb6cbda7e) ([merge request](gitlab-org/gitlab-pages!725))

## 1.56.1 (2022-03-28)

### Changed (1 change)

- [Update go-proxyproto to 0.6.2 and fix tests](gitlab-org/security/gitlab-pages@4c8e257183d9fe8684de3b90787176582bcc8298) ([merge request](gitlab-org/security/gitlab-pages!23))

### Security (2 changes)

- [fix: validate that session was issued on the same host](gitlab-org/security/gitlab-pages@e0b2a7070c3398e74aeeba2c4cc249bf0eb689bf) ([merge request](gitlab-org/security/gitlab-pages!29))
- [Fix weak HTTP server timeouts configuration](gitlab-org/security/gitlab-pages@fc5a652574d0eef03c776a70f3c0678158bc1dbd) ([merge request](gitlab-org/security/gitlab-pages!19))

## 1.56.0 (2022-03-15)

### Added (3 changes)

- [feat: allow auth http.Client timeout to be configurable](gitlab-org/gitlab-pages@0a2122d4960ebdca71a21cdb6038696f1746c3f1) by @Osmanilge ([merge request](gitlab-org/gitlab-pages!687))
- [feat: make server shutdown timeout configurable](gitlab-org/gitlab-pages@f78d8d18b960f66a2a4f4e2044e2159647d375af) by @HuseyinEmreAksoy ([merge request](gitlab-org/gitlab-pages!688))
- [Add security-harness script](gitlab-org/gitlab-pages@de0b946ff919a2df3e172c569383dec8a4fd3b41) ([merge request](gitlab-org/gitlab-pages!697))

## 1.55.0 (2022-02-22)

### Added (1 change)

- [feat: Add TLS rate limits](gitlab-org/gitlab-pages@62a6491652aa6975d9ecf3b9e258766c886d49d4) ([merge request](gitlab-org/gitlab-pages!700))

### Fixed (1 change)

- [fix: do no retry resolving the domain if there's a ctx error](gitlab-org/gitlab-pages@970531c7f80db47d209196921043aabcdf7590ef) by @feistel ([merge request](gitlab-org/gitlab-pages!691))

## 1.54.2 (2022-04-13)

### Fixed (1 change)

- [Increase serverWriteTimeout to avoid errors with large files](gitlab-org/gitlab-pages@61dd377fa1b63d3498a7cc9e1c09959a4ca52090) ([merge request](gitlab-org/gitlab-pages!724))

## 1.54.1 (2022-03-28)

### Changed (1 change)

- [Update go-proxyproto to 0.6.2 and fix tests](gitlab-org/security/gitlab-pages@9154a5e7e98ec5492503f49c3ad28da6cfc3043b) ([merge request](gitlab-org/security/gitlab-pages!25))

### Security (2 changes)

- [fix: validate that session was issued on the same host](gitlab-org/security/gitlab-pages@5806a2d13e7c3354c477162329c21aa6377af70f) ([merge request](gitlab-org/security/gitlab-pages!30))
- [Fix weak HTTP server timeouts configuration](gitlab-org/security/gitlab-pages@8bd2398e301877a98f8efe3738861a7d96b87d7f) ([merge request](gitlab-org/security/gitlab-pages!20))

## 1.54.0 (2022-02-10)

### Fixed (1 change)

- [fix: ensure logging status codes field names are consistent](gitlab-org/gitlab-pages@6f23e35ffe9665ab17af54824a7de2b014829069) ([merge request](gitlab-org/gitlab-pages!679))

## 1.53.0 (2022-02-01)

### Fixed (2 changes)

- [fix: Fix 500 errors when clients disconnect](gitlab-org/gitlab-pages@1b50e38f3959c784f44c280720ed3249802d2622) ([merge request](gitlab-org/gitlab-pages!681))
- [fix: fix metrics and logs not including domain resolution time](gitlab-org/gitlab-pages@adc0b9233fa4a0b2b449e27336d9ae39c75819ba) ([merge request](gitlab-org/gitlab-pages!674))

### Changed (1 change)

- [refactor: stop passing file descriptors around and use net.Listen](gitlab-org/gitlab-pages@052a7fb36f4605634385f54c833db21a9edc6d67) by @feistel ([merge request](gitlab-org/gitlab-pages!667))

## 1.52.0 (2022-01-31)

### Added (1 change)

- [feat: implement graceful shutdown](gitlab-org/gitlab-pages@b2b9c8d346f96257cd0fbc9577df1b9e81c28d21) by @feistel ([merge request](gitlab-org/gitlab-pages!664))

### Fixed (1 change)

- [fix: log errors when HTTP range requests fail](gitlab-org/gitlab-pages@42bae52a4a5d439efe77244d3105daf4f3598b62) ([merge request](gitlab-org/gitlab-pages!675))

### Changed (1 change)

- [feat: switch to content negotiation library](gitlab-org/gitlab-pages@3f287da18498c8f98855ba43484b712026685d9c) by @feistel ([merge request](gitlab-org/gitlab-pages!624))

## 1.51.2 (2022-04-13)

### Fixed (1 change)

- [Increase serverWriteTimeout to avoid errors with large files](gitlab-org/gitlab-pages@df9c7c413e35940feb7b1ea3664cb6f6e03814a3) ([merge request](gitlab-org/gitlab-pages!723))

## 1.51.1 (2022-03-28)

### Changed (1 change)

- [Update go-proxyproto to 0.6.2 and fix tests](gitlab-org/security/gitlab-pages@4dc370158507bd8dd64e1ca1f451c870c0fb56e6) ([merge request](gitlab-org/security/gitlab-pages!26))

### Security (2 changes)

- [fix: validate that session was issued on the same host](gitlab-org/security/gitlab-pages@8c7fe1f00874ea94161570c040136c1b1a53d3a2) ([merge request](gitlab-org/security/gitlab-pages!31))
- [Fix weak HTTP server timeouts configuration](gitlab-org/security/gitlab-pages@80f38361c88eb5d23132528f1d528acd47ab5a18) ([merge request](gitlab-org/security/gitlab-pages!21))

## 1.51.0 (2022-01-12)

### Added (2 changes)

- [feat: add domain rate-limiter](gitlab-org/gitlab-pages@f62bc0dd08e3a9e14b6bf6c35b04dd6bdcec4491) ([merge request](gitlab-org/gitlab-pages!635))
- [feat: enable Etag caching](gitlab-org/gitlab-pages@d40af81155fdd50f7ea3b78249cc303b91acf63e) ([merge request](gitlab-org/gitlab-pages!653))

### Other (1 change)

- [chore: Use Go v1.16.12](gitlab-org/gitlab-pages@e971baaace956056f0477a23f5e03406f9f9296f) ([merge request](gitlab-org/gitlab-pages!660))

## 1.50.0 (2021-12-29)

No changes.

## 1.49.0 (2021-12-16)

### Added (1 change)

- [feat: handle extra headers when serving from compressed zip archive](gitlab-org/gitlab-pages@85621f69f855e43afe983e2ca107e921aa14c8c8) by @feistel ([merge request](gitlab-org/gitlab-pages!529))

### Fixed (2 changes)

- [feat: hide handling cache headers behind the faeture flag](gitlab-org/gitlab-pages@0cfd85f8edd8f52db96b72b9efb99f0183053536) ([merge request](gitlab-org/gitlab-pages!645))
- [fix(auth): check suffix correctly in domainAllowed](gitlab-org/gitlab-pages@50095a1ceda366e5ed6b7adfe72d3387c44d1be8) by @mlegner ([merge request](gitlab-org/gitlab-pages!619))

### Changed (2 changes)

- [fix: update vfs/zip implementation to ensure minimum range requests for go1.17](gitlab-org/gitlab-pages@26e1d310b513cc235f00336f8866c8c059f7ce80) ([merge request](gitlab-org/gitlab-pages!646))
- [refactor: replace deprecated StandardClaims with RFC7519-compliant RegisteredClaims](gitlab-org/gitlab-pages@1d0f0e81ba7db826d42051705e89ba304ad95ce7) by @feistel ([merge request](gitlab-org/gitlab-pages!608))

### Other (1 change)

- [chore: upgrade to labkit 1.11.0](gitlab-org/gitlab-pages@86d8aac645d1a6ccb24ab57c87e4aacf535bec7b) ([merge request](gitlab-org/gitlab-pages!633))

### changed. (1 change)

- [fix(vfs): handle context.Canceled errors](gitlab-org/gitlab-pages@d9c27aea72fb9ef1dcfe3c8fb4ecface8400e534) ([merge request](gitlab-org/gitlab-pages!628))

## 1.48.0 (2021-11-15)

- [chore: Update golang to 1.16.10](gitlab-org/gitlab-pages@d016cf6a8ac3c1569ee4e41317dbd76a2ef5a1ef) ([merge request](gitlab-org/gitlab-pages!615))

## 1.47.0 (2021-11-11)

### Fixed (1 change)

- [fix: reject requests with very long URIs](gitlab-org/gitlab-pages@bf9c79a5477b61f375be659e2e16f377067d9c00) ([merge request](gitlab-org/gitlab-pages!612))

### change (1 change)

- [refactor: read and use file_sha256 as cache key for zip VFS](gitlab-org/gitlab-pages@77a733b8af393f81c1143ca27324384b1801a090) by @feistel ([merge request](gitlab-org/gitlab-pages!527))

## 1.46.0 (2021-10-14)

### Added (2 changes)

- [feat: add source IP ratelimiter middleware](gitlab-org/gitlab-pages@ccfdff303646b86daed2bd9ae7e2f2a5eb4a2c5c) ([merge request](gitlab-org/gitlab-pages!594))
- [feat: add ratelimiter package](gitlab-org/gitlab-pages@49f2c8908c0d831e6a2880c8ad659116ad70d74d) ([merge request](gitlab-org/gitlab-pages!587))

### Removed (1 change)

- [refactor: stop running gitlab-pages as root](gitlab-org/gitlab-pages@dc7d694f00eadd078a05991bff7c78cb29efeff4) by @feistel ([merge request](gitlab-org/gitlab-pages!542))

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
