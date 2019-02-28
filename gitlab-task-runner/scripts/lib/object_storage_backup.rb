require 'open3'
require 'fileutils'

class String
  def red; "\e[31m#{self}\e[0m" end
  def green; "\e[32m#{self}\e[0m" end
  def blue; "\e[34m#{self}\e[0m" end
end

class ObjectStorageBackup
  attr_accessor :name, :local_tar_path, :remote_bucket_name, :tmp_bucket_name, :backend_type

  def initialize(name, local_tar_path, remote_bucket_name, tmp_bucket_name = 'tmp', backend_type = 's3')
    @name = name
    @local_tar_path = local_tar_path
    @remote_bucket_name = remote_bucket_name
    @tmp_bucket_name = tmp_bucket_name
    @backend_type = backend_type
  end

  def backup
    puts "Dumping #{@name} ...".blue

    if @backend_type == "s3"
      cmd = %W(s3cmd sync s3://#{@remote_bucket_name} /srv/gitlab/tmp/#{@name})
    elsif @backend_type == "gcs"
      cmd = %W(gsutil rsync gs://#{@remote_bucket_name} /srv/gitlab/tmp/#{@name})
    end

    output, status = run_cmd(cmd)
    failure_abort(output) unless status.zero?

    return unless File.exist? "/srv/gitlab/tmp/#{@name}" # Bucket may be empty
    cmd = %W(tar -czf #{@local_tar_path} -C /srv/gitlab/tmp/#{@name} . )
    output, status = run_cmd(cmd)
    failure_abort(output) unless status.zero?

    puts "done".green
  end

  def restore
    puts "Restoring #{@name} ...".blue

    backup_existing
    cleanup
    restore_from_backup
    puts "done".green
  end

  def failure_abort(error_message)
    puts "[Error] #{error_message}".red
    abort "Restore #{@name} failed"
  end

  def upload_to_object_storage(source_path)
    if @backend_type == "s3"
      # s3cmd treats `-` as a special filename for using stdin, as a result
      # we need a slightly different syntax to support syncing the `-` directory (used for system uploads)
      if File.basename(source_path) == '-'
        cmd = %W(s3cmd sync #{source_path}/ s3://#{@remote_bucket_name}/-/)
      else
        cmd = %W(s3cmd sync #{source_path} s3://#{@remote_bucket_name})
      end
    elsif @backend_type == "gcs"
      cmd = %W(gsutil rsync #{source_path}/ gs://#{@remote_bucket_name})
    end

    output, status = run_cmd(cmd)

    failure_abort(output) unless status.zero?
  end

  def backup_existing
    backup_file_name = "#{@name}.#{Time.now.to_i}"

    if @backend_type == "s3"
      cmd = %W(s3cmd sync s3://#{@remote_bucket_name} s3://#{@tmp_bucket_name}/#{backup_file_name}/)
    elsif @backend_type == "gcs"
      cmd = %W(gsutil rsync gs://#{@remote_bucket_name} gs://#{@tmp_bucket_name}/#{backup_file_name}/)
    end

    output, status = run_cmd(cmd)

    failure_abort(output) unless status.zero?
  end

  def cleanup
    if @backend_type == "s3"
      cmd = %W(s3cmd del --force --recursive s3://#{@remote_bucket_name})
    elsif @backend_type == "gcs"
      cmd = %W(gsutil rm -f -r gs://#{@remote_bucket_name})
    end
    output, status = run_cmd(cmd)
    failure_abort(output) unless status.zero?
  end

  def restore_from_backup
    extracted_tar_path = File.join(File.dirname(@local_tar_path), "/srv/gitlab/tmp/#{@name}")
    FileUtils.mkdir_p(extracted_tar_path, mode: 0700)

    failure_abort("#{@local_tar_path} not found") unless File.exist?(@local_tar_path)

    untar_cmd = %W(tar -xf #{@local_tar_path} -C #{extracted_tar_path})

    output, status = run_cmd(untar_cmd)

    failure_abort(output) unless status.zero?

    Dir.glob("#{extracted_tar_path}/*").each do |file|
     upload_to_object_storage(file)
    end
  end

  def run_cmd(cmd)
    _, stdout, wait_thr = Open3.popen2e(*cmd)
    return stdout.read, wait_thr.value.exitstatus
  end

end
