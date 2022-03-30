# Getting started with development

## Configuring gitlab-pages hostname

gitlab-pages needs a hostname/domain, since each different pages sites is accessed via a
subdomain. gitlab-pages hostname can be set in different manners, like:

### Without wildcard, editing `/etc/hosts`

Since `/etc/hosts` don't support wildcard hostnames, you'll have to add to your configuration one
entry for the gitlab-pages and then one entry for each page site.

   ```text
   127.0.0.1 gdk.test           # If you're using GDK
   127.0.0.1 pages.gdk.test     # Pages host
   # Any namespace/group/user needs to be added
   # as a subdomain to the pages host. This is because
   # /etc/hosts doesn't accept wildcards
   127.0.0.1 root.pages.gdk.test # for the root pages
   ```

### With dns wildcard alternatives

If instead of editing your `/etc/hosts` you'd prefer to use a dns wildcard, you can use:

- [`nip.io`](https://nip.io)
- [`dnsmasq`](https://wiki.debian.org/dnsmasq)

## Configuring gitlab-pages

Create a `gitlab-pages.conf` in the root of the gitlab-pages site, like:

```toml
listen-http=:3010             # default port is 3010, but you can use any other
pages-domain=pages.gdk.test   # your local gitlab-pages domain
pages-root=shared/pages       # directory where the pages are stored
log-verbose=true              # show more information in the logs
```

To see more options you can check [`internal/config/flags.go`](https://gitlab.com/gitlab-org/gitlab-pages/blob/master/internal/config/flags.go)
or run `gitlab-pages --help`.

## Running gitlab-pages with GDK

In the following steps, `$GDK_ROOT` is the directory where you cloned GDK.

1. Set up the [gdk hostname](https://gitlab.com/gitlab-org/gitlab-development-kit/-/blob/main/doc/howto/local_network.md).
1. Add a [gitlab-pages hostname](#configuring-gitlab-pages-hostname) to the `gdk.yml`:

   ```yaml
   gitlab_pages:
     enabled: true
     host: pages.gdk.test
   ```

1. Set the gdk related configuration:

   ```config
   listen-http=:3010                              # default port
   gitlab-server=http://gdk.test:3000             # gitlab server
   artifacts-server=http://gdk.test:3000/api/v4   # gitlab api base url
   pages-domain=pages.gdk.test                    # gitlab-pages hostname
   pages-root=$GDK_ROOT/gitlab/shared/pages       # directory where the pages are stored
   api-secret-key=$GDK_ROOT/gitlab-pages-secret   # GDK access tokens
   ```

1. Once these configurations are set GDK will manage a gitlab-pages process and you'll have access
   to it with commands like:

   ```sh
   $ gdk start gitlab-pages   # start gitlab-pages
   $ gdk stop gitlab-pages    # stop gitlab-pages
   $ gdk restart gitlab-pages # restart gitlab-pages
   $ gdk tail gitlab-pages    # tail gitlab-pages logs
   ```

### Running gitlab-pages manually

You can also build and start the app independent of GDK processes management.

For any changes in the code, you must run `make` to build the app, so it's best to just always run
it before you start the app. It's quick to build so don't worry!

```sh
make && ./gitlab-pages -config=gitlab-pages.conf
```

To build in FIPS mode

```sh
$ FIPS_MODE=1 make && ./gitlab-pages -config=gitlab-pages.conf
```

### Creating gitlab-pages site

To build a gitlab-pages site locally you'll have to [configure `gitlab-runner`](https://gitlab.com/gitlab-org/gitlab-development-kit/-/blob/main/doc/howto/runner.md)

Check the [user manual](https://docs.gitlab.com/ee/user/project/pages/).

### Enable access control

gitlab-pages have support to private sites, which means sites that only people who has access to the
Gitlab's project will have access to its gitlab-pages site.

gitlab-pages access control is disabled by default. To enable it:

1. Enable the gitlab-pages access control within gitlab itlsef, which can be done by editing
   `gitlab.yml` or in the `gdk.yml` if you're using GDK.

   ```yaml
   # gitlab/config/gitlab.yml
   pages:
     access_control: true
   ```

   or

   ```yaml
   # $GDK_ROOT/gdk.yml
   gitlab_pages:
     enabled: true
     access_control: true
   ```

1. Restart GitLab (if running through the GDK, run `gdk restart`). Note that running
   `gdk reconfigure` overwrites the value of `access_control` in `config/gitlab.yml`.
1. In your local GitLab instance, in the browser navigate to `http://gdk.test:3000/admin/applications`.
1. Create an [Instance-wide OAuth application](https://docs.gitlab.com/ee/integration/oauth_provider.html#instance-wide-applications).
   - The scope is `api`
1. Set the value of your `redirect-uri` to the `pages-domain` authorization endpoint
   - `http://pages.gdk.test:3010/auth`, for example
   - Note that the `redirect-uri` must not contain any gitlab-pages site domain
1. Add these lines to your `gitlab-pages.conf` file:

   ```conf
   ## the following are only needed if you want to test auth for private projects
   auth-client-id=$CLIENT_ID                          # the OAuth application id created in http://gdk.test:3000/admin/applications
   auth-client-secret=$CLIENT_SECRET                  # the OAuth application secret created in http://gdk.test:3000/admin/applications
   auth-secret=$SOME_RANDOM_STRING                    # should be at least 32 bytes long
   auth-redirect-uri= http://pages.gdk.test:3010/auth # the authentication callback url for gitlab-pages
   ```

1. If running Pages inside the GDK, you can add the `gitlab-pages.conf` file to the
   `protected_config_files` section under `gdk` in your `gdk.yml` to avoid getting these
   configuration rewritten by GDK:

   ```yaml
   gdk:
     protected_config_files:
     - 'gitlab-pages/gitlab-pages.conf'
   ```

## Linting

```sh
# Get everything installed and setup (you only need to run this once)
# If you run into problems running the linting process,
# you may have to run `make clean` and try this step again
make setup

# Run the linter locally
make lint
```

## Testing

To run tests, you can use these commands:

```sh
# This will run all of the tests in the codebase
make test

# Run a specfic test file
go test ./internal/serving/disk/

# Run a specific test in a file
go test ./internal/serving/disk/ -run TestDisk_ServeFileHTTP

# Run all unit tests except acceptance_test.go
go test ./... -short

# Run acceptance_test.go only
make acceptance
# Run specific acceptance tests
# We add `make` here because acceptance tests use the last binary that was compiled,
# so we want to have the latest changes in the build that is tested
make && go test ./ -run TestRedirect
```
