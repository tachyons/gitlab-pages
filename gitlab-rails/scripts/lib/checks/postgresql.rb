require 'yaml'
require 'active_record'

class Checks
    class PostgreSQL
        @@config = nil
        @@database_version = 0
        
        def self.wait_for_timeout()
            ENV["WAIT_FOR_TIMEOUT"].to_i
        end

        def self.sleep_duration()
            ENV["SLEEP_DURATION"].to_i
        end

        def self.codebase_schema_version()
            ENV["SCHEMA_VERSION"].to_i
        end

        def self.config()
            return @@config if @@config
            config = YAML.load_file("#{ENV["CONFIG_DIRECTORY"]}/#{ENV["DATABASE_FILE"]}")
            @@config = config['production']
        end

        def self.database_schema_version()
            # load ActiveRecord with configuration
            ActiveRecord::Base.establish_connection(config)
            # Attempt to fetch the current migrations version
            begin
                @@database_version = ActiveRecord::Migrator.current_version
                true
            rescue PG::ConnectionBad => e
                puts "PostgreSQL Error: #{e.message}"
                false
            rescue RuntimeError => e
                puts "Error: #{e.message}"
                false
            end
        end

        def self.check_schema_version()
            success = database_schema_version

            puts "Database Schema - current: #{@@database_version}, codebase: #{codebase_schema_version}"
            
            puts "NOTICE: Database has not been initialized yet." unless @@database_version.to_i > 0

            return true unless ENV["BYPASS_SCHEMA_VERSION"].nil? and success

            return (@@database_version <= codebase_schema_version)
        end

        def self.run()
            counter = 1
            passed = false
            until counter == wait_for_timeout or passed do
                passed = check_schema_version
                sleep sleep_duration unless passed
                counter+=1
            end
            passed
        end
    end
end
