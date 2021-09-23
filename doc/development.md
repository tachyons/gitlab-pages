# Getting started with development

If you want to develop GitLab Pages with the GDK, follow [these instructions](https://gitlab.com/gitlab-org/gitlab-development-kit/-/blob/main/doc/howto/pages.md).

You can also run and develop GitLab Pages outside of the GDK. Here are a few commands and host file
changes to get you running with the examples built into the repository.

Create `gitlab-pages.conf` in the root of this project:

```
# Replace `192.168.1.135` with your own local IP
pages-domain=192.168.1.135.nip.io
pages-root=shared/pages
listen-http=:8090
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

1. Modify your `config/gitlab.yml` file:

   ```rb
   pages:
     access_control: true
   ```

1. Restart GitLab (if running through the GDK, run `gdk restart`). Note that running
   `gdk reconfigure` overwrites the value of `access_control` in `config/gitlab.yml`.

1. In your local GitLab instance, navigate to `/admin/applications`.
1. Create an [OAuth application](https://docs.gitlab.com/ee/integration/oauth_provider.html#add-an-application-through-the-profile).
1. Set the value of your `redirect-uri` to the `pages-domain` authorization endpoint (for example
   `http://192.168.1.135.nip.io:8090/auth`).
1. Add these lines to your `gitlab-pages.conf` file:

   ```conf
   ## the following are only needed if you want to test auth for private projects
   auth-client-id=$CLIENT_ID # generate a new OAuth application in http://127.0.0.1:3000/admin/applications
   auth-client-secret=$CLIENT_SECRET # obtained when generating an OAuth application
   auth-secret= $SOME_RANDOM_STRING # should be at least 32 bytes long
   auth-redirect-uri=http://192.168.1.135.nip.io:8090/auth
   ```

1. If running Pages inside the GDK, you can add the `gitlab-pages.conf` file to the
   `protected_config_files` section under `gdk` in your `gdk.yml` file:

   ```yaml
   gdk:
     protected_config_files:
     - 'gitlab-pages/gitlab-pages.conf'
   ```

## Developing inside the GDK

This is an example of developing GitLab Pages inside the [GitLab Development Kit (GDK)](https://gitlab.com/gitlab-org/gitlab-development-kit):

1. [Prepare your GDK environment](https://gitlab.com/gitlab-org/gitlab-development-kit#how-to-install-gdk).
   In the steps that follow, `$GDK_ROOT` is the directory where you cloned the GDK.
1. Add the following lines to your `gdk.yml` file:

   ```yaml
   # You can use dnsmasq to use a different hostname https://www.tecmint.com/setup-a-dns-dhcp-server-using-dnsmasq-on-centos-rhel/
   hostname: 127.0.0.1.nip.io
   gitlab_pages:
     auto_update: true
     enabled: true
     port: 3010
     secret_file: $GDK_ROOT/gitlab-pages-secret # run make gitlab-pages-secret in your $GDK_ROOT
     verbose: true
     host: pages.127.0.0.1.nip.io

   # enable Object Storage to use the latest features
   object_store:
     enabled: true
     port: 9000

   # only needed if you are using ssh
   repositories:
     gitlab_pages: git@gitlab.com:gitlab-org/gitlab-pages.git

   # add this line to keep changes to your gitlab-pages.conf file intact after running `gdk reconfigure`
   gdk:
     protected_config_files:
     - 'gitlab-pages/gitlab-pages.conf'

   sshd:
     enabled: true
     listen_port: 2222
     user: your-uuser
   ```

1. Reconfigure the GDK by running `gdk reconfigure`.
1. Go to `$GDK_ROOT/gitlab-pages`:

   ```sh
   cd $GDK_ROOT/gitlab-pages
   ```

   Note that running `gdk reconfigure` overrides your `gitlab-pages.conf` file and sets the default
   flags. Make sure you add the file to the `protected_config_files:` YAML node in your `gdk.yml`
   file.

1. Create or edit the file `$GDK_ROOT/gitlab-pages/gitlab-pages.conf` to add these lines:

   ```conf
   # the port where you want Pages to listen to, must match the port in `gdk.yml`
   listen-http=:3010
   artifacts-server=http://127.0.0.1.nip.io:3000/api/v4
   # absolute path inside $GDK_ROOT
   pages-root=$GDK_ROOT/gitlab/shared/pages
   pages-domain=pages.127.0.0.1.nip.io
   internal-gitlab-server=http://127.0.0.1.nip.io:3000
   gitlab-server=http://127.0.0.1.nip.io:3000
   # run make gitlab-pages-secret in your $GDK_ROOT
   api-secret-key=$GDK_ROOT/gitlab-pages-secret
   log-verbose=true
   ## the following settings are only needed if you want to test auth for private projects
   auth-client-id=$CLIENT_ID # generate a new OAuth application in http://127.0.0.1.nip.io:3000/admin/applications
   auth-client-secret=$CLIENT_SECRET # obtained when generating an OAuth application
   auth-secret= $SOME_RANDOM_STRING # should be at least 32 bytes long
   auth-redirect-uri=http://pages.127.0.0.1.nip.io:3010/auth
   ```

   You can define any flags available in [`main.go`](https://gitlab.com/gitlab-org/gitlab-pages/-/blob/ec16301b72b5d8370ccdcd86088440cca409cd8b/main.go#L40).

1. Start developing!
1. To test your changes manually you can run:

   ```sh
   # Inside $GDK_ROOT/gitlab-pages
   $ make
   $ gdk restart gitlab-pages
   $ gdk tail gitlab-pages

   # or one-liner
   make && gdk restart gitlab-pages && gdk tail gitlab-pages
   ```

1. Alternatively, you can run Pages manually:

   ```sh
   # Inside $GDK_ROOT/gitlab-pages
   $ gdk stop gitlab-pages
   $ make # calls go build in this project and creates a `gitlab-pages` binary under bin/
   # start daemon manually with a config
   $ ./bin/gitlab-pages -config gitlab-pages.conf
   ```

1. Create a project in your GDK and deploy a Pages project. For instructions, see
   [Create a GitLab Pages website from scratch](https://docs.gitlab.com/ee/user/project/pages/getting_started/pages_from_scratch.html).
1. To deploy your Pages site, you must [configure GitLab Runner in your GDK](https://gitlab.com/gitlab-org/gitlab-development-kit/-/blob/master/doc/howto/runner.md).
1. Visit your project URL. You can see the URL under **Settings > Pages** for your project, or
   [`http://127.0.0.1.nip.io:3000/user/project-name/pages`](http://127.0.0.1.nip.io:3000/user/project-name/pages).

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
