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
        passed = check_schema_version
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

    def self.codebase_schema_version
      ENV['SCHEMA_VERSION'].to_i
    end

    def self.config_directory
      ENV['CONFIG_DIRECTORY']
    end

    def self.database_file
      ENV['DATABASE_FILE']
    end

    def self.config
      return @@config if @@config

      config = YAML.load_file(File.join(config_directory, database_file))
      @@config = config['production']
    end

    def self.database_schema_version
      ActiveRecord::Base.establish_connection(config)
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

    def self.check_schema_version
      success = database_schema_version

      puts "Database Schema - current: #{@@database_version}, codebase: #{codebase_schema_version}"

      puts 'NOTICE: Database has not been initialized yet.' unless @@database_version.to_i.positive?

      return true unless ENV['BYPASS_SCHEMA_VERSION'].nil? && success

      (@@database_version <= codebase_schema_version)
    end
  end
end
