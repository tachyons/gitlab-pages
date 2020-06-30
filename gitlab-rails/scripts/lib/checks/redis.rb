# frozen_string_literal: true

require 'yaml'
require 'redis'
# Get Hash.deep_symbolize_keys from Rails
# - allows transparent transformation of Sentinel configuration
require 'active_support/core_ext/hash/keys'

module Checks
  # Perform checks of Redis dependency.
  # Usage: `Checks::Redis.run`
  module Redis
    def self.run
      counter = 1
      passed = false
      until (counter == wait_for_timeout) || passed
        passed = check_redis_connectivity
        sleep sleep_duration unless passed
        counter += 1
      end
      passed
    end

    def self.wait_for_timeout
      ENV['WAIT_FOR_TIMEOUT'].to_i
    end

    def self.sleep_duration
      ENV['SLEEP_DURATION'].to_i
    end

    def self.redis_config(file)
      resque_yaml = YAML.load_file(file)
      resque_yaml['production'].deep_symbolize_keys
    end

    def self.ping_redis(file)
      redis = ::Redis.new(redis_config(file))
      url = "#{redis._client.scheme}://#{redis._client.host}:#{redis._client.port}"
      begin
        redis.ping
      rescue RuntimeError => e
        puts "- FAILED connecting to '#{url}' from #{file}, through #{redis._client.host}\r\nERROR: #{e.message}"
        false
      else
        redis.disconnect!
        puts "+ SUCCESS connecting to '#{url}' from #{file}, through #{redis._client.host}"
        true
      end
    end

    def self.check_redis_connectivity
      Dir.chdir(ENV['CONFIG_DIRECTORY'])
      files = Dir.glob(['resque.yml', "redis\..*.yml", 'cable.yml'])
      puts "Checking: #{files.join(', ')}"
      results = files.map do |resque_file|
        Thread.new { Redis.ping_redis(resque_file) }
      end.map(&:value)

      # Collect the checks that passed.
      checks_passed = results.select { |r| r }
      # Return pass/fail based on all checks passing
      checks_passed.count == files.count
    end
  end
end
