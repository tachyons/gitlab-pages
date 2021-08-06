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

    class DatabaseConfig
      def initialize(shard_name)
        @shard_name = shard_name
      end

      def check_schema_version
        ActiveRecord::Base.connected_to(shard: @shard_name, role: :writing) do
          success = database_schema_version

          puts "Database Schema - #{@shard_name} (#{ActiveRecord::Base.connection_db_config.database}) - current: #{@database_version}, codebase: #{codebase_schema_version}"

          puts 'NOTICE: Database has not been initialized yet.' unless @database_version.to_i.positive?

          return true if (ENV['BYPASS_SCHEMA_VERSION'] && success)

          (success && @database_version.to_i >= codebase_schema_version)
        end
      rescue => e
        puts "Error checking #{@shard_name}: #{e.message}"
        false
      end

      private

      def database_schema_version
        begin
          @database_version = ActiveRecord::Base.connection.migration_context.current_version

          # Rails silently eats `ActiveRecord::NoDatabaseError` when calling `current_version`
          # This stems from https://github.com/rails/rails/blob/v6.0.3.6/activerecord/lib/active_record/connection_adapters/postgresql_adapter.rb#L48-L54
          puts "WARNING: Problem accessing #{@shard_name} database (#{ActiveRecord::Base.connection_db_config.database})."\
               " Confirm username, password, and permissions." if @database_version.nil?

          # returning false prevents bailing when BYPASS_SCHEMA_VERSION set.
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

    def self.check_all_databases
      ActiveRecord::Base.legacy_connection_handling = false
      ActiveRecord::Base.configurations = database_configurations

      production_databases = database_configurations.configs_for(
        env_name: 'production', include_replicas: false)

      puts "Checking: #{production_databases.map(&:name).join(', ')}"

      results = production_databases.map do |db_config|
        ActiveRecord::Base.connection_handler.establish_connection(
          db_config, role: :writing, shard: db_config.name)

        Thread.new do
          DatabaseConfig.new(db_config.name).check_schema_version
        end
      end.map(&:value)

      # Collect the checks that passed.
      results.all?
    end
  end
end
