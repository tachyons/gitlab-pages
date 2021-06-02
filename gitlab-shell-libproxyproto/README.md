### gitlab-shell with PROXY support

This image is based off the gitlab-shell image but adds [PROXY protocol](https://developers.cloudflare.com/spectrum/proxy-protocol) support via [libproxyproto](https://github.com/msantos/libproxyproto).

The Debian and Ubuntu patches to support this can be found in [this
repository](https://gitlab.com/gitlab-com/gl-infra/openssh-patches).

#### Configuration varaibles

See [the list of environment variables](https://github.com/msantos/libproxyproto#environment-variables)
that can be used.

#### Quick start

To enforce, PROXY v2 protocol, set:

```yaml
LIBPROXYPROTO_MUST_USE_PROTOCOL_HEADER: 1
LIBPROXYPROTO_VERSION: 2
```

To test this image with debug logging:

```sh
docker run -e LIBPROXYPROTO_MUST_USE_PROTOCOL_HEADER=1 -e LIBPROXYPROTO_DEBUG=1 -it -v /run/sshd:/run/sshd -p 2222:2222 registry.gitlab.com/gitlab-org/build/cng/gitlab-shell-libproxyproto
```

This will start up an OpenSSH server with PROXY support on port 2222.
