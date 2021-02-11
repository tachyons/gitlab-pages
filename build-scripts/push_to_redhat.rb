#!/usr/bin/env ruby

# The push_to_redhat.rb script will retag a number of container images
# and push them to Red Hat for certification tests. Each image has a
# unique ID and a pull secret used to access the Red Hat registry. These
# values are found in the REDHAT_SECRETS_JSON variable with is an encoded
# JSON string.

require 'json'
require 'digest'

$GITLAB_REGISTRY = ENV['GITLAB_REGISTRY_BASE_URL'] || 'registry.gitlab.com/gitlab-org/build/cng'
$REDHAT_REGISTRY = ENV['REDHAT_REGISTRY_HOSTNAME'] || 'scan.connect.redhat.com'
$CONTAINER_NAMES = [ 'alpine-certificates',
                     'gitaly',
                     'gitlab-container-registry',
                     'gitlab-exporter',
                     'gitlab-mailroom',
                     'gitlab-rails-ee',
                     'gitlab-shell',
                     'gitlab-sidekiq-ee',
                     'gitlab-task-runner-ee',
                     'gitlab-webservice-ee',
                     'gitlab-workhorse-ee',
                     'kubectl' ]

def retag_image(name, version, proj_id)
  gitlab_tag = "#{version}-ubi8"
  redhat_tag = version.gsub(/^v(\d+\.\d+\.\d+)/, '\1')
  new_container_name = "#{$REDHAT_REGISTRY}/#{proj_id}/#{name}:#{redhat_tag}"

  puts "Retagging #{$GITLAB_REGISTRY}/#{name}:#{gitlab_tag} to #{new_container_name}"
  %x(docker tag #{$GITLAB_REGISTRY}/#{name}:#{gitlab_tag} #{new_container_name})
  new_container_name
end

def set_credentials(secret)
  puts "Setting credentials"
  puts "checksum = #{Digest::SHA1.hexdigest secret}"
  %x(echo '#{secret}' | docker login -u unused --password-stdin #{$REDHAT_REGISTRY})
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
    container_name = retag_image(name, version, secrets[name]['id'])

    result = set_credentials(secrets[name]['secret']).chomp
    if result != 'Login Succeeded'
      puts "***** Failed to authenticate to registry for #{name} *****"
      puts "#{result}\n"
      errors << "#{name}: Unable to authentcate to registry (bad secret?)"
      next
    end

    puts push_image(container_name)
  else
    puts "No entry for #{name} in secrets file"
    errors << "#{name}: No secret listed in $REDHAT_SECRETS_JSON"
  end
end

unless errors.empty?
  puts "\n\nThe following errors have been collected:"
  errors.each { |err|
    puts "\t- #{err}"
  }
  exit(1)
end
