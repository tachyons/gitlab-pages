# frozen_string_literal: true

require 'yaml'
require 'active_record'

module Checks
  # Perform checks of PostgreSQL dependency.
  # Usage: `Checks::PostgreSQL.run`
  module PostgreSQL
    @@config = nil
    @@database_version = 0

    def self.run
      counter = 1
      passed = false
      until (counter == wait_for_timeout) || passed
        passed = check_all_databases
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

    def self.config_directory
      ENV['CONFIG_DIRECTORY']
    end

    def self.database_file
      ENV['DATABASE_FILE']
    end

    def self.db_schema_target
      ENV['DB_SCHEMA_TARGET']
    end

    class DatabaseConfig
      def initialize(shard_name)
        @shard_name = shard_name
      end

      def check_schema_version
        success = database_schema_version

        puts "Database Schema - #{@shard_name} (#{ActiveRecord::Base.connection_db_config.database}) - current: #{@database_version}, codebase: #{codebase_schema_version}"
        puts 'NOTICE: Database has not been initialized yet.' unless @database_version.to_i.positive?

        return true if (ENV['BYPASS_SCHEMA_VERSION'] && success)

        (success && @database_version.to_i >= codebase_schema_version)
      rescue => e
        puts "Error checking #{@shard_name}: #{e.message}"
        false
      end

      private

      def database_schema_version
        begin
          db_config = ActiveRecord::Base.configurations.configs_for(env_name: 'production', name: @shard_name)
          connection = ActiveRecord::Base.establish_connection(db_config).connection
          schema_migrations_table_name = ActiveRecord::Base.schema_migrations_table_name

          if connection.table_exists?(schema_migrations_table_name)
            @database_version =
              connection.execute("SELECT MAX(version) AS version FROM #{schema_migrations_table_name}")
                        .first
                        .fetch('version')
          end

          puts "WARNING: Problem accessing #{@shard_name} database (#{ActiveRecord::Base.connection_db_config.database})."\
               " Confirm username, password, and permissions." if @database_version.nil?

          # Returning false prevents bailing when BYPASS_SCHEMA_VERSION set.
          !@database_version.nil?
        rescue RuntimeError => e
          puts "Error fetching #{@shard_name} schema: #{e.message}"
          false
        end
      end

      def codebase_schema_version
        # TODO: This may be suspect if there is a separate schema per shard
        ENV['SCHEMA_VERSION'].to_i
      end
    end

    def self.database_configurations
      @@database_configurations ||= ActiveRecord::DatabaseConfigurations
        .new(database_yaml)
    end

    def self.database_yaml
      @@database_yaml ||= ActiveSupport::ConfigurationFile.parse(
        File.join(config_directory, database_file))
    end

    def self.production_databases
      db_configs = database_configurations.configs_for(
        env_name: 'production', include_replicas: false)

      db_configs =
        if db_schema_target == 'geo'
          # TODO: To be removed in 15.0. See https://gitlab.com/gitlab-org/gitlab/-/issues/351946
          # The db_config.name is set to primary when config/database_geo.yml exists and uses a legacy syntax.
          db_configs.find { |db_config| ['primary', 'geo'].include?(db_config.name) }
        else
          db_configs.reject { |db_config| db_config.name == 'geo' }
        end

      Array(db_configs)
    end

    def self.check_all_databases
      ActiveRecord::Base.legacy_connection_handling = false
      ActiveRecord::Base.configurations = database_configurations

      puts "Checking: #{production_databases.map(&:name).join(', ')}"

      results = production_databases.map do |db_config|
        Thread.new do
          DatabaseConfig.new(db_config.name).check_schema_version
        end
      end.map(&:value)

      # Collect the checks that passed.
      results.all?
    end
  end
end
