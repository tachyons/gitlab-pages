require 'yaml'
require 'redis'
# Get Hash.deep_symbolize_keys from Rails
# - allows transparent transformation of Sentinel configuration
require 'active_support/core_ext/hash/keys'


class Checks
    class Redis
        def self.wait_for_timeout()
            ENV["WAIT_FOR_TIMEOUT"].to_i
        end

        def self.sleep_duration()
            ENV["SLEEP_DURATION"].to_i
        end

        def self.ping_redis(file)
            begin
                resque_yaml = YAML.load_file(file)
                config = resque_yaml['production'].deep_symbolize_keys
                # configure the Redis client
                redis = ::Redis.new( config )
                url = "#{redis._client.scheme}://#{redis._client.host}:#{redis._client.port}"
                puts "> Connecting to '#{url}' from #{file}"
                # "ping" the server
                redis.ping
            rescue RuntimeError => e
                puts "- FAILED connecting to '#{url}' from #{file}, through #{redis._client.host}"
                puts "  ERROR: #{e.message}"
                false
            else
                # nicely disconnect
                redis.disconnect!
                puts "+ SUCCESS connecting to '#{url}' from #{file}, through #{redis._client.host}"
                true
            end
        end

        def self.check_redis_connectivity()
            Dir.chdir(ENV["CONFIG_DIRECTORY"])
            files = Dir.glob(["resque.yml", "redis\..*.yml", "cable.yml"])
            puts "Checking: #{files.join(", ")}"
            results = files.map do |resque_file|
                Thread.new { Redis.ping_redis(resque_file) }
            end.map(&:value)

            # Collect the checks that passed.
            checks_passed = results.select {|r| r }
            # Return pass/fail based on all checks passing
            checks_passed.count == files.count
        end

        def self.run()
            counter = 1
            passed = false
            until counter == wait_for_timeout or passed do
                passed = check_redis_connectivity
                sleep sleep_duration unless passed
                counter+=1
            end
            passed
        end
    end
end