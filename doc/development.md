# Getting started with development

If you want to develop GitLab Pages with the GDK, follow [these instructions](https://gitlab.com/gitlab-org/gitlab-development-kit/blob/master/doc/howto/pages.md).

You can also run and develop GitLab Pages outside of the GDK. Here is a few commands and host file changes to get you running with the examples built into the repo.

Create `gitlab-pages.conf` in the root of this project:

```
# Replace `192.168.1.135` with your own local IP
pages-domain=192.168.1.135.nip.io
pages-root=shared/pages
listen-http=:8090
# WARNING: to be deprecated in https://gitlab.com/gitlab-org/gitlab-pages/-/issues/382
domain-config-source=disk
log-verbose=true
```

Build and start the app. For any changes, you must run `make` to build the app, so it's best to just always run it before you start the app. It's quick to build so don't worry!

```sh
make && ./gitlab-pages -config=gitlab-pages.conf
```

Visit http://group.192.168.1.135.nip.io:8090/project/index.html (replace `192.168.1.135` with your IP) and you should see a
`project-subdir` response

You can see our [testing](#testing) and [linting](#linting) sections below on how to run those.

### I don't want to use `nip.io`

If you don't want to use `nip.io` for the wildcard DNS, you can use one of these methods.

A simple alternative is to add a `/etc/hosts` entry pointing from `localhost`/`127.0.0.1` to the directory subdomain for any directory under `shared/pages/`.
This is because `/etc/hosts` does not support wildcard hostnames.

```
127.0.0.1 pages.gdk.test
# You will need to an entry for every domain/group you want to access
127.0.0.1 group.pages.gdk.test
```

An alternative is to use [`dnsmasq`](https://wiki.debian.org/dnsmasq) to handle wildcard hostnames.


### Enable access control

Pages access control is disabled by default. To enable it:

1. Modify your config/gitlab.yml file:

```rb
pages:
  access_control: true
```

2. Restart GitLab (if running via GDK run: `gdk restart`)

**NOTE**:
Running `gdk reconfigure` will overwrite the value of `access_control` in `config/gitlab.yml`.

3. In your local GitLab instance, navigate to `/admin/applications`
4. Create an [OAuth application](https://docs.gitlab.com/ee/integration/oauth_provider.html#add-an-application-through-the-profile)
5. Set the value of your redirect-uri to the `pages-domain` authorization endpoint, for example `http://192.168.1.135.nip.io:8090/auth`
6. Add the following lines to your `gitlab-pages.conf` file

```conf
## the following are only needed if you want to test auth for private projects
auth-client-id=$CLIENT_ID # generate a new OAuth application in http://127.0.0.1:3000/admin/applications
auth-client-secret=$CLIENT_SECRET # obtained when generating an OAuth application
auth-secret= $SOME_RANDOM_STRING # should be at least 32 bytes long
auth-redirect-uri=http://192.168.1.135.nip.io:8090/auth
```

7. If running Pages inside the GDK, you can add the `gitlab-pages.conf` file to the `protected_config_files` section under `gdk` in your `gdk.yml` file

```yaml
gdk:
  protected_config_files:
  - 'gitlab-pages/gitlab-pages.conf'
```

## Linting

```sh
# Get everything installed and setup (you only need to run this once)
# If you run into problems running the linting process,
# you may have to run `sudo rm -rf .GOPATH` and try this step again
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
