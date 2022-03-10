#!/usr/bin/env ruby
# frozen_string_literal: true

# The redhat_certification.rb script will invoke the Red Hat API to request
# an image to be scanned for certification. Each image has an associated
# project ID used to identify the Red Hat certification project. These
# values are found in the REDHAT_PROJECT_JSON variable which is an encoded
# JSON string.

require 'json'
require 'yaml'
require 'digest'
require 'uri'
require 'net/http'
require 'optparse'

def gitlab_registry
  ENV['GITLAB_REGISTRY_BASE_URL'] || ENV['CI_REGISTRY_IMAGE'] ||
    'registry.gitlab.com/gitlab-org/build/cng'
end

def regular_tag?
  (ENV['CI_COMMIT_TAG'] || ENV['GITLAB_TAG'])
end

def redhat_api_endpoint
  'https://catalog.redhat.com/api/containers/v1/projects/certifications/requests/scans'
end

def read_project_data
  git_root = `git rev-parse --show-toplevel`.strip
  YAML.load_file("#{git_root}/redhat-projects.yaml")
rescue StandardError => e
  puts "Unable to parse #{git_root}/redhat-projects.yaml: #{e.message}"
  raise
end

# return either the sha256 of the image or the symbol :no_skopeo
def image_sha256(registry_spec)
  skopeo_out = `skopeo inspect docker://#{registry_spec} 2>/dev/null`
  JSON.parse(skopeo_out)['Digest']
rescue Errno::ENOENT
  :no_skopeo
rescue JSON::ParserError
  :no_image
end

# handle a call the Red Hat API and return the response object
def redhat_api(method, endpoint, token = nil, payload = nil)
  uri = URI.parse(endpoint)
  http = Net::HTTP.new(uri.host, uri.port)
  http.use_ssl = true

  case method
  when :post
    req = Net::HTTP::Post.new(uri.request_uri)
  when :get
    req = Net::HTTP::Get.new(uri.request_uri)
  else
    raise ArgumentError, "Unknown method (#{method}) for redhat_api function"
  end

  req.add_field('Content-Type', 'application/json')
  req.add_field('X-API-KEY', token)
  req.body = payload.to_json if method == :post

  begin
    resp = http.request(req)
  rescue StandardError => e
    puts "Unhandled exception for #{name}: #{e}"
    errors << "#{name}: Unhandled exception: #{e}"
  end

  resp
end

def display_scan_status(token)
  results = redhat_api(:get,
                       redhat_api_endpoint,
                       token)

  # invert the GitLab container name to Red Hat proj ID hash
  rh_pid = {}
  read_project_data.each_pair { |name, ids| rh_pid[ids['pid']] = name }

  puts ' Status     Red Hat Project ID          Request ID                  GitLab Project:Tag'
  puts '=======    ========================    ========================    ==================================='

  stats = { pending: 0, running: 0 }
  JSON.parse(results.body)['data'].each do |scan_req|
    puts "#{scan_req['status']}    #{scan_req['cert_project']}    " \
         "#{scan_req['_id']}    #{rh_pid[scan_req['cert_project']]}:#{scan_req['tag']}"
    case scan_req['status']
    when 'pending'
      stats[:pending] += 1
    when 'running'
      stats[:running] += 1
    end
  end

  puts "\nTotal pending: #{stats[:pending]}\t\tTotal running: #{stats[:running]}"
end

options = { status: false, token: ENV['REDHAT_API_TOKEN'] }
OptionParser.new do |opts|
  opts.banner = "Usage: #{File.basename $PROGRAM_NAME} [options] "

  opts.on('-s', '--status', 'Get current image scan request status') do
    options[:status] = true
  end

  opts.on('-t', '--token TOKEN', 'Red Hat API token') do |val|
    options[:token] = val
  end
end.parse!

if options[:status]
  display_scan_status(options[:token])
  exit(0)
end

if ARGV.empty?
  puts 'Need to specify a version (i.e. v13.5.4)'
  exit 1
end

# Remove CE/EE suffix and add `-ubi8` to the commit ref if not already present.
version = ARGV[0].sub(/-(ce|ee)$/, '')
unless version.end_with? '-ubi8'
  # we add `-ubi8` to find the UBI images.
  version += '-ubi8'
end

puts "Using #{version} as the docker tag to pull"

errors = []
project_data = read_project_data
project_data.each_key do |name|
  # if job is on a tagged pipeline (but not a auto-deploy tag) or
  # is a master branch pipeline, then use the image tags as
  # defined in variables defined in the CI environment. Otherwise
  # it is assumed that the "version" (commit ref) from CLI param
  # is correct.
  if ENV['CI_COMMIT_REF_NAME'] == 'master' || regular_tag?
    version = "#{ENV[project_data[name]['version_variable']].sub(/-(ce|ee)$/, '')}-ubi8"
  end

  if project_data.key? name
    sha256_tag = image_sha256("#{gitlab_registry}/#{name}:#{version}")
    case sha256_tag
    when :no_skopeo
      errors << 'skopeo command is not installed'
      next
    when :no_image
      errors << "Image with #{version} tag not found for #{name}"
      next
    end

    endpoint = "https://catalog.redhat.com/api/containers/v1/projects/certification/id/#{project_data[name]['pid']}/requests/scans"
    payload = {
      'pull_spec' => "#{gitlab_registry}/#{name}@#{sha256_tag}",
      'tag'       => version
    }
    resp = redhat_api(:post, endpoint, options[:token], payload)

    puts "API call for #{name} returned #{resp.code}: #{resp.message}"
    if resp.code.to_i < 200 || resp.code.to_i > 299
      errors << "API call for #{name} returned #{resp.code}: #{resp.message}"
    end
  else
    # let someone know that there was not a secret for a specific image
    puts "No entry for #{name} in redhat-projects.yaml"
    errors << "#{name}: No project info listed in redhat-projects.yaml"
  end
end

# display the collected errors in the CI job output
unless errors.empty?
  puts "\n\nThe following errors have been collected:"
  errors.each do |err|
    puts "\t- #{err}"
  end
  exit(1)
end
