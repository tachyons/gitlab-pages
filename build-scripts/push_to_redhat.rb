#!/usr/bin/env ruby

# The push_to_redhat.rb script will retag a number of container images
# and push them to Red Hat for certification tests. Each image has a
# unique ID and a pull secret used to access the Red Hat registry. These
# values are found in the REDHAT_SECRETS_JSON variable with is an encoded
# JSON string.

require 'json'
require 'digest'
require 'uri'
require 'net/http'

$GITLAB_REGISTRY = ENV['GITLAB_REGISTRY_BASE_URL'] || ENV['CI_REGISTRY_IMAGE'] || 'registry.gitlab.com/gitlab-org/build/cng'
$REDHAT_REGISTRY = ENV['REDHAT_REGISTRY_HOSTNAME'] || 'scan.connect.redhat.com'
$IMAGE_VERSION_VAR = { 'alpine-certificates'       => 'ALPINE_VERSION',
                       'gitaly'                    => 'GITALY_SERVER_VERSION',
                       'gitlab-container-registry' => 'GITLAB_CONTAINER_REGISTRY_VERSION',
                       'gitlab-exporter'           => 'GITLAB_EXPORTER_VERSION',
                       'gitlab-mailroom'           => 'MAILROOM_VERSION',
                       'gitlab-shell'              => 'GITLAB_SHELL_VERSION',
                       'gitlab-sidekiq-ee'         => 'GITLAB_VERSION',
                       'gitlab-toolbox-ee'         => 'GITLAB_VERSION',
                       'gitlab-webservice-ee'      => 'GITLAB_VERSION',
                       'gitlab-workhorse-ee'       => 'GITLAB_VERSION',
                       'kubectl'                   => 'KUBECTL_VERSION' }


def is_regular_tag
  (ENV['CI_COMMIT_TAG'] || ENV['GITLAB_TAG']) && \
  !($AUTO_DEPLOY_BRANCH_REGEX.match(ENV['CI_COMMIT_BRANCH']) || $AUTO_DEPLOY_TAG_REGEX.match(ENV['CI_COMMIT_TAG']))
end

if ARGV.length < 1
  puts "Need to specify a version (i.e. v13.5.4)"
  exit 1
end

# Remove CE/EE suffix and add `-ubi8` to the commit ref if not already present.
version = ARGV[0].sub(/-(ce|ee)$/, '')
if not version.end_with? '-ubi8'
  # we add `-ubi8` to find the UBI images.
  version += '-ubi8'
end

# pull in the secrets used to auth with Red Hat registries (CI var)
begin
  secrets = JSON.parse(ENV['REDHAT_SECRETS_JSON'])
rescue => e
  puts "Unable to parse JSON: #{e.message}"
  puts e.backtrace
  raise
end

puts "Using #{version} as the docker tag to pull"

errors = []
$IMAGE_VERSION_VAR.keys.each do |name|
  # if job is on a tagged pipeline (but not a auto-deploy tag) or
  # is a master branch pipeline, then use the image tags as
  # defined in variables defined in the CI environment. Otherwise
  # it is assumed that the "version" (commit ref) from CLI param
  # is correct.
  if (ENV['CI_COMMIT_REF_NAME'] == 'master' || is_regular_tag)
    version = ENV[$IMAGE_VERSION_VAR[name]].sub(/-(ce|ee)$/, '') + '-ubi8'
  end

  if secrets.has_key? name
    endpoint = "https://catalog.redhat.com/api/containers/v1/projects/certification/id/#{secrets[name]['id']}/requests/scans"
    uri = URI.parse(endpoint)
    http = Net::HTTP.new(uri.host, uri.port)
    http.use_ssl = true
    req = Net::HTTP::Post.new(uri.request_uri)
    req.add_field('Content-Type', 'application/json')
    req.add_field('X-API-KEY', ENV['REDHAT_API_TOKEN'])
    payload = { 'pull_spec' => 'registry.gitlab.com/.....',
                'tag'       => version }
    req.body = payload.to_json

    begin
      resp = http.request(req)
    rescue Exception => e
      puts "Unhandled exception for #{name}: #{e}"
      errors << "#{name}: Unhandled exception: #{e}"
    end
  else
    # let someone know that there was not a secret for a specific image
    puts "No entry for #{name} in secrets file"
    errors << "#{name}: No secret listed in $REDHAT_SECRETS_JSON"
  end
end

# display the collected errors in the CI job output
unless errors.empty?
  puts "\n\nThe following errors have been collected:"
  errors.each { |err|
    puts "\t- #{err}"
  }
  exit(1)
end
