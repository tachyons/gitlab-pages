#!/usr/bin/env ruby

# The push_to_redhat.rb script will retag a number of container images
# and push them to Red Hat for certification tests. Each image has a
# unique ID and a pull secret used to access the Red Hat registry. These
# values are found in the REDHAT_SECRETS_JSON variable with is an encoded
# JSON string.

require 'json'

$GITLAB_REGISTRY = 'registry.gitlab.com/gitlab-org/build/cng'
$REDHAT_REGISTRY = 'scan.connect.rehat.com'
$CONTAINER_NAMES = ['kubectl', 'gitlab-workhorse-ee', 'gitlab-webservice-ee',
                    'gitlab-task-runner-ee', 'gitlab-sidekiq-ee', 'gitlab-shell',
                    'gitlab-rails-ee', 'gitlab-mailroom', 'gitlab-exporter',
                    'gitlab-container-registry', 'gitaly', 'alpine-certificates', ]

def tag_image(name, version, proj_id)
  gitlab_tag = "#{version}-ubi8"
  redhat_tag = version.gsub(/^v/, '')
  new_container_name = "#{$REDHAT_REGISTRY}/#{proj_id}/#{name}:#{redhat_tag}"

  puts "Retagging #{$GITLAB_REGISTRY}/#{name}:#{gitlab_tag} to #{new_container_name}"
  %x(docker tag #{$GITLAB_REGISTRY}/#{name}:#{gitlab_tag} #{new_container_name})
  new_container_name
end

def set_credentials(secret)
  puts "Setting credentials"
  #%x(echo #{secret} | docker login -u unused --password-stdin scan.connect.redhat.com)
  %x(docker login -u unused -p #{secret} scan.connect.redhat.com)
end

def pull_image(image)
  puts "Pulling #{image}-ubi8"
  %x(docker pull #{image}-ubi8)
end

def push_image(image)
  puts "Pushing #{image}"
  %x(docker push #{image})
end

if ARGV.length < 1
  puts "Need to specify a version (i.e. v13.5.4)"
  exit 1
end

version = ARGV[0]
begin
  secrets = JSON.parse(ENV['REDHAT_SECRETS_JSON'])
rescue => e
  puts "Unable to parse JSON: #{e.message}"
  puts e.backtrace
  raise
end

puts "Using #{version}-ubi8 as the docker tag to pull"

errors = []
$CONTAINER_NAMES.each do |name|
  if secrets.has_key? name
    response = pull_image("#{$GITLAB_REGISTRY}/#{name}:#{version}")
    if response.empty?
      puts "Skipping #{$GITLAB_REGISTRY}/#{name}:#{version}-ubi8 (Not Found)"
      errors << "#{name}: image not found with #{version}-ubi8 tag"
      next
    end

    # retag the image with the Red Hat registry information
    container_name = tag_image(name, version, secrets[name]['id'])

    result = set_credentials(secrets[name]['pull_secret'])
    if result != 'Login Succeeded'
      puts "***** Failed to authenticate to registry for #{name} *****"

      errors << "#{name}: Unable to authentcate to registry (bad pull secret?)"
      next
    end

    puts push_image(container_name)
  else
    puts "No entry for #{name} in secrets file"
    errors << "#{name}: No pull secret listed in $REDHAT_SECRETS_JSON"
  end
end

unless errors.empty?
  puts "\n\nThe following errors have been collected:"
  errors.each { |err|
    puts "\t- #{err}"
  }
  exit(1)
end
