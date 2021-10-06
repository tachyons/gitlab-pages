#!/usr/bin/env ruby

# The redhat_certification.rb script will invoke the Red Hat API to request
# an image to be scanned for certification. Each image has an associated
# project ID used to identify the Red Hat certification project. These
# values are found in the REDHAT_PROJECT_JSON variable which is an encoded
# JSON string.

require 'json'
require 'digest'
require 'uri'
require 'net/http'
require 'optparse'

$GITLAB_REGISTRY = ENV['GITLAB_REGISTRY_BASE_URL'] || ENV['CI_REGISTRY_IMAGE'] || 'registry.gitlab.com/gitlab-org/build/cng'
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

def read_project_data
  begin
    JSON.parse(ENV['REDHAT_PROJECT_JSON'])
  rescue => e
    puts "Unable to parse JSON: #{e.message}"
    puts e.backtrace
    raise
  end
end

# return either the sha256 of the image or the symbol :no_skopeo
def image_sha256(registry_spec)
  begin
    skopeo_out = %x(skopeo inspect docker://#{registry_spec} 2>/dev/null)
    JSON.parse(skopeo_out)['Digest']
  rescue Errno::ENOENT
    return :no_skopeo
  rescue JSON::ParserError
    return :no_image
  end
end

#
def redhat_api(method, endpoint, token=nil, payload=nil)
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
  rescue Exception => e
    puts "Unhandled exception for #{name}: #{e}"
    errors << "#{name}: Unhandled exception: #{e}"
  end

  return resp
end


def display_scan_status(token)
  results = redhat_api(:get,
              'https://catalog.redhat.com/api/containers/v1/projects/certifications/requests/scans',
              token=token)

  # invert the GitLab container name to Red Hat proj ID hash
  rh_pid = {}
  read_project_data.each_pair { |name,ids| rh_pid[ids['pid']] = name }

  stats = {:pending => 0, :running => 0}
  JSON.parse(results.body)['data'].each do |scan_req|
    puts "#{scan_req['status']}\t\t#{rh_pid[scan_req['cert_project']]}:#{scan_req['tag']}"
    case scan_req['status']
    when 'pending'
      stats[:pending] += 1
    when 'running'
      stats[:running] += 1
    end
  end
  puts ""
  puts "Total pending: #{stats[:pending]}\t\tTotal running: #{stats[:running]}"
end


token = ENV.keys.include?('REDHAT_API_TOKEN') ? ENV['REDHAT_API_TOKEN'] : nil
options = {:status => false, :token => token }
optparse = OptionParser.new do |opts|
  opts.banner = "Usage: #{File.basename $0} [options] "

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


puts "Using #{version} as the docker tag to pull"

errors = []
project_data = read_project_data()
$IMAGE_VERSION_VAR.keys.each do |name|
  # if job is on a tagged pipeline (but not a auto-deploy tag) or
  # is a master branch pipeline, then use the image tags as
  # defined in variables defined in the CI environment. Otherwise
  # it is assumed that the "version" (commit ref) from CLI param
  # is correct.
  if (ENV['CI_COMMIT_REF_NAME'] == 'master' || is_regular_tag)
    version = ENV[$IMAGE_VERSION_VAR[name]].sub(/-(ce|ee)$/, '') + '-ubi8'
  end

  if project_data.has_key? name
    sha256_tag = image_sha256("#{$GITLAB_REGISTRY}/#{name}:#{version}")
    case sha256_tag
    when :no_skopeo
      errors << "skopeo command is not installed"
      next
    when :no_image
      errors << "Image with #{version} tag not found for #{name}"
      next
    end

    endpoint = "https://catalog.redhat.com/api/containers/v1/projects/certification/id/#{project_data[name]['pid']}/requests/scans"
    payload = { 'pull_spec' => "#{$GITLAB_REGISTRY}/#{name}@#{sha256_tag}",
                'tag'       => version }
    resp = redhat_api(:post, endpoint, token=token, payload=payload)


    puts "API call for #{name} returned #{resp.code}: #{resp.message}"
    if resp.code.to_i < 200 or resp.code.to_i > 299
      errors << "API call for #{name} returned #{resp.code}: #{resp.message}"
    end
  else
    # let someone know that there was not a secret for a specific image
    puts "No entry for #{name} in CI variable REDHAT_PROJECT_JSON"
    errors << "#{name}: No project info listed in $REDHAT_PROJECT_JSON"
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
