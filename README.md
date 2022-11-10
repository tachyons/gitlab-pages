# Cloud Native GitLab container images (CNG)

## Overview

Cloud Native GitLab build pipelines leverage caching and shared layers for efficiency. This approach
enables shorter build times and smaller images. It also reduces operational overhead through shared
layers which reduce the bandwidth required during initial installation or deploying upgrades.

For all components to be included for customer deployments, a directory of a
component shall be created.  It is advisable to reach out to Distribution as
early in the process as possible to ensure effective and timely support prior to
enabling a new component to be made available for customer consumption.

The mechanisms created in this repo manage dependencies required by any
service as well as ensure conformance to various standards such as the
Federal Information Processing Standards (FIPS). This removes the burden
of image building from development teams and enables standardization for all images.

Most images are based on [Debian Linux](https://debian.org). A few smaller, quick
running tasks are based on the [official container image](https://hub.docker.com/_/alpine/) from [Alpine Linux image](https://alpinelinux.org/).

Each directory contains the `Dockerfile` for a specific component of the
infrastructure needed to run GitLab.

* [rails](/gitlab-rails) - The Rails code needed for both API and web.
* [webservice](/gitlab-webservice) - The webservice container that exposes Rails webservers (Puma).
* [workhorse](/gitlab-workhorse) - The GitLab Workhorse container providing smart-proxy in front of Rails.
* [sidekiq](/gitlab-sidekiq) - The Sidekiq container that runs async Rails jobs.
* [shell](/gitlab-shell) - Running GitLab Shell and OpenSSH to provide git over ssh, and authorized keys support from the database
* [gitaly](/gitaly) - The Gitaly container that provides a distributed git repos
* [gitlab-kas](/gitlab-kas) - The backend for the GitLab Agent for Kubernetes
* [toolbox](/gitlab-toolbox) - The toolbox container provides utilities for direct interaction with the application suite, without interrupting service containers.

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

## Container Design Philosophy

A quick review of one of the larger component builds illuminates why intentional use
of additional layers and intermediate build images creates smaller final images.

### Caching

Cloud Native GitLab container pipelines first build optimized layers common to all container builds. The
job runners in the build pipelines share these layers via Docker's build caching mechanisms. This
pattern decreases overall build times.

### Layering

Cloud Native GitLab containers also reduce image sizes through the use of shared layers. The final set
of image containers share common layers reducing overall footprint. Shared layers also decrease operating expenses (OpEx) for the administrator through reduced bandwidth during upgrades and storage requirements for GitLab deployments. The details of these savings are discussed further in a later section.

#### Dependencies

Many container images rely on the same subset of dependencies.  For example, the
base layer builds on top of Debian and installs standard build tools. The build
pipeline caches this  [`gitlab-base`] layer and reuses it during later build
jobs.

The pipeline builds containers for common language frameworks such as ruby
and golang. Each GitLab component then builds using common versions and tooling
which ensures consistent results across the application. The respective containers
are [`gitlab-ruby`] and [`gitlab-go`].

#### Final Images

The final images add any specific requirements on top of the cached shared layers.
For example, [`Mailroom`] uses the [`gitlab-ruby`] image but does not require any
of the rails codebase. The [`gitlab-rails`] image bundles the rails codebase with
the [`gitlab-ruby`] image for sharing with other containers such as [`gitlab-webservice`].
This separation continues the pattern of decreased container sizes and build times in
downstream container builds.

<details><summary>Click here for a functional example</summary>

By using `docker inspect`, we can gain a better view into how so many layers
comprise our images and with high amount of reuse, we can minimize how much
bandwidth is needed overall to download images.  

```diff
% diff --color -U 40 \
  <(docker inspect registry.gitlab.com/gitlab-org/build/cng/gitlab-webservice-ee:v15.4.0 | jq .[].RootFS.Layers[]) \
  <(docker inspect registry.gitlab.com/gitlab-org/build/cng/gitlab-sidekiq-ee:v15.4.0 | jq .[].RootFS.Layers[])
@@ -1,32 +1,32 @@
 "sha256:b45078e74ec97c5e600f6d5de8ce6254094fb3cb4dc5e1cc8335fb31664af66e"
 "sha256:fd643292238bc5d32420c4e03b3927bdabebf99de964031d59723e7d874eca40"
 "sha256:dd411100902b3fa371997a24ed092c9dc3d470c1e8898bf839ef56929ae3b961"
 "sha256:40df3d36d899fc90896bc63f5881f9714cf2409f00a377a592d7540af8987351"
 "sha256:627b4e8525669b36942b3e5c97461f945fc7b7e22251309d661235f926f09daf"
 "sha256:f3870b341eec6f2696ea86a1f17d564928d2e915aecbcf5026cea40a79fb58d9"
 "sha256:985fc94066d453bead85cf836b55be683fb123e233eba61f762eef56b7e8c021"
 "sha256:577247be027c7fa8e8b9db331066232660b76fdaac3c8611535ded694d68a51c"
 "sha256:bf05b68c8a2e992ea410df3bd656660ce4ad3726ec54c5bcfdaf11b337787fa9"
 "sha256:12ce81e0f2570f7fbcd742fcf6428eaa4e5644b831fbf94c879710c998421bf1"
 "sha256:f399dd0c7ac65715d97dfff7b2a708eb91e73a45e4bfc911572528e264bc2d6e"
 "sha256:6c006e5770ab50b677b1103cb6c053ae58e0dffd49ab47b8c0c6a61d86072410"
 "sha256:c787e47d78e38ba1cf710cf4895bb53b33cd3e0db38c0b5747347723e85b3b3c"
 "sha256:17b1392502ca7a57ebccf8c5b302ad18563ffb6a65fe861fb71d74f65266b930"
 "sha256:3cdc2bdf92e945f266db986bdebdea23acd355abe91d0725fbc93d50dbd581ef"
 "sha256:39d483041b76c66c2722b7beacfb30a195b66ec1559b885c43661543ea7c8dc2"
 "sha256:43231816729198ec05577ab9a1bf235227e279a9b181142dfe1129fcaf31a6c4"
 "sha256:a413935fbc91e5df215b24cc5e1e23c5aa1d4cfa6162eb5da02ba33e3da6aabf"
 "sha256:f51a98ce19ec45d5db4a8136acc676a259e0bcefc705e3cd9156806a8b1cbf7f"
 "sha256:af643e37a5c997ab994ab4e0d104d02bd787ea72e1fe0415486a53f816c83905"
 "sha256:50bb97783119f4e95b321ec3a98555d4e45f0d9456d3846e5dfccf4b971b752b"
 "sha256:dddc3d64b78dd4a69bcf1936858a6a756d5a72c6ac907ba3b8d3b69263de00cb"
 "sha256:861f978252b899bcbfeecc314e2c6945451b3e6f24471153aba86f4c8f40be61"
 "sha256:f033f58ab76d06ca0f1b6584fe9aa98a6b62701b00f26d488fbc233194782158"
 "sha256:76bdbd30178ae9d82a1b9c61b135a1cca2f13681b607290bd6014a8bfed5b8d4"
-"sha256:9ddae558702ea3c415481edc02ac2b0f87fc0884a60b12a0ae842e42b2de2d8e"
-"sha256:f5d7eae67c981e48028aa8f4da8ca8610efa703c9514f7924e329d6187696b30"
-"sha256:a6186b0a60ba43a6ae8f0b68822f70c585df3604e457c23536c348b391957328"
-"sha256:07bba2c84b6ff89631ef835bd19b84f9457dd5de676596c71d3371d740f807d4"
-"sha256:2560f9ccbcfe1c46076e1aa961a78bea74e7985628d241b6007c9b26b9ce40c4"
-"sha256:61d0670e99720714b86a2f84d024bed6e5e8a4103ba2ceff2484cc64b26db1f5"
-"sha256:5d2b2ed789a68f3f162f0c12f77770544a9afa656a14384ba5c6861d8732436f"
+"sha256:4e33539734ec9496bffec045848e86b784b5d94d8e1b348c6ced5ad0408a8a71"
+"sha256:fef185332358f4b2cbadf6760e42948de2cf2a84b46e115aa1343a181e356d83"
+"sha256:5cd75fe83d31adab55498e0a2bcc60e180a86f0571c95d4b1be2a2dd66e4ff82"
+"sha256:78af44038bfb7606a47a9471e54095a2b256ada50c8745f657042a6cfc8e85bb"
+"sha256:958b28453b33a0100970e1a95edeed0cfdffff7dea3722e513c0868886801c37"
+"sha256:a70ad27ca0507dd5e965e0d026d916ed7afb751a3d5b7cbddc1980e0a73ec71b"
+"sha256:885c865ed6e69e06be1d856232b5f09a998b9575e0d33b9283327d922d6a82d8"
```

Out of the 32 layers each of these images consists of, the first 25 are reused.
This is 25 individual layers that only need to be downloaded once for the same
version of GitLab that may be pulled down.  So what does this savings look like?

```
% docker manifest inspect \
  registry.gitlab.com/gitlab-org/build/cng/gitlab-sidekiq-ee:v15.4.0 | jq '[.layers[0:25] | .[] | .size] | add'
617574719
```

So we've effectively combined roughly 600MB of data into the first 25 layers
that is reused in a few images.  So how much data is not reused between the
webservice and sidekiq containers?

```
% docker manifest inspect \
  registry.gitlab.com/gitlab-org/build/cng/gitlab-sidekiq-ee:v15.4.0 | jq '[.layers[25:32] | .[] | .size] | add'
53834608
% docker manifest inspect \
  registry.gitlab.com/gitlab-org/build/cng/gitlab-webservice-ee:v15.4.0 | jq '[.layers[25:32] | .[] | .size] | add'
37880086
```

This translates into roughly 54MB in sidekiq, and 40MB for our webservice
containers.  The end result of this heavy use of layering results in a single 600MB
download + a few tens of MB for our additional images.

</details>

##### End Result

The end result of this layering scheme is a complex tree of dependencies that require careful
consideration.  Most layers are duplicated so that when a customer
pulls both `gitlab-sidekiq` and `gitlab-webservice` the common shared layers
only need to be downloaded once per node.  For each
image, every attempt is made to avoid rebuilding when possible.  Each build phase
checks to determine if a build is required based on criterion configured in the
build pipeline.  Since the rails codebase is compiled in a higher stage than
most final images, and since some dependencies we build into layers that are
higher in the chain, there's a higher chance of some images being built
unnecessarily.  There are various efforts to increase the efficiency further
than what exists today:

* [Improve Contributor Experience in
  CNG](https://gitlab.com/groups/gitlab-org/-/epics/6692)
* [Improve Distribution Pipelines to increase Velocity and
  Productivity](https://gitlab.com/groups/gitlab-org/-/epics/5746)
* [Distribution Long term Build Efficiency
  Vision](https://gitlab.com/groups/gitlab-org/-/epics/6679)

#### Configuration

Support for configuration is intended to be as follows:

1. Mounting templates for the config files already supported by our different software (gitlab.yml, database.yml, resque.yml, etc)
2. Additionally support the environment variables supported by the software, like https://docs.gitlab.com/ce/administration/environment_variables.html (support them by not doing anything that would drop them from being passed to the running process)
3. Add ENV variables for configuring the custom code we use in the containers, like the rendering in or of templates, and any wrapper/helper commands

Templating languages supported:

1. [ERB](https://docs.ruby-lang.org/en/2.7.0/ERB.html), following traditional standards (`<% %>`) will be available in all Ruby-based application containers.
2. [gomplate](https://docs.gomplate.ca/), using `{% %}` non-standard delimiters (ensuring compatibility with Helm's internal use of `{{ }}`) will be available in all GitLab originated containers.
    - NOTE: [datasource](https://docs.gomplate.ca/datasources/) usage via `-d` is not supported. For more advanced usage see the gomplate [functions](https://docs.gomplate.ca/syntax/#functions).

> For Kubernetes specifically we are mostly relying on the mounting the config
files from ConfigMap objects. With the occasional ENV variable to control the
custom container code.

## Links

1. [Building Images](docs/build.md)

[`gitlab-base`]: https://gitlab.com/gitlab-org/build/CNG/-/blob/master/gitlab-base/Dockerfile
[`gitlab-rails`]: https://gitlab.com/gitlab-org/build/CNG/-/blob/master/gitlab-rails/Dockerfile
[`gitlab-go`]: https://gitlab.com/gitlab-org/build/CNG/-/tree/master/gitlab-go/Dockerfile
[`gitlab-ruby`]: https://gitlab.com/gitlab-org/build/CNG/-/blob/master/gitlab-ruby/Dockerfile
[`Mailroom`]: https://gitlab.com/gitlab-org/build/CNG/-/blob/master/gitlab-mailroom/Dockerfile
[`gitlab-webservice`]: https://gitlab.com/gitlab-org/build/CNG/-/blob/master/gitlab-webservice/Dockerfile
