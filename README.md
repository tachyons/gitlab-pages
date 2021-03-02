### Cloud native GitLab Docker Images

Most images are based on the [Debian Linux](https://debian.org), using the [official buster-slim image](https://hub.docker.com/_/debian). A few smaller, quick running tasks are based on [Alpine Linux](https://alpinelinux.org/), using the [official image](https://hub.docker.com/_/alpine/).

Built using [the official source installation instructions](http://docs.gitlab.com/ee/install/installation.html) with some Alpine specific fixes
and some dependency compilation tweaks picked up from from the [omnibus build packages](https://gitlab.com/gitlab-org/omnibus-gitlab).

Each directory contains the `Dockerfile` for a specific component of the
infrastructure needed to run GitLab.

* [rails](/gitlab-rails) - The Rails code needed for both API and web.
* [webservice](/gitlab-webservice) - The webservice container that exposes Rails webservers (Unicorn or Puma).
* [workhorse](/gitlab-workhorse) - The GitLab Workhorse container providing smart-proxy in front of Rails.
* [sidekiq](/gitlab-sidekiq) - The Sidekiq container that runs async Rails jobs.
* [shell](/gitlab-shell) - Running GitLab Shell and OpenSSH to provide git over ssh, and authorized keys support from the database
* [gitaly](/gitaly) - The Gitaly container that provides a distributed git repos
* [task-runner](/gitlab-task-runner) - The task runner container provides utilities for direct interaction with the application suite, without interruping service containers.

### Dev environment using Docker Compose

A dev test environment is provided with docker-compose.

To run the environment:

```bash
# Grab the latest Images
docker-compose pull
# Start GitLab
docker-compose up
```

The instance should then be reachable at `http://localhost:3000`

#### Registry access

As the `docker-compose` deployment does not make use of TLS, `docker` will
be "unhappy". To address this, you can add the following to
`/etc/docker/daemon.json` and then restart the service. It will allow
any hostname that resolves to `127.0.0.1` to be handled as insecure.

```json
{
  "insecure-registries" : [ "127.0.0.1" ]
}
```

### Design of the Containers

#### Configuration

Support for configuration is intended to be as follows:

1. Mounting templates for the config files already supported by our different software (gitlab.yml, database.yml, resque.yml, etc)
2. Additionally support the environement variables supported by the software, like https://docs.gitlab.com/ce/administration/environment_variables.html (support them by not doing anything that would drop them from being passed to the running process)
3. Add ENV variables for configuring the custom code we use in the containers, like the the ERB rendering in the templates, and any wrapper/helper commands


> For Kubernetes specifically we are mostly relying on the mounting the config
files from ConfigMap objects. With the occasional ENV variable to control the
custom container code.

### Links

1. [Building Images](docs/build.md)
