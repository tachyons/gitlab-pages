# Relative URL support
# WARNING: We recommend using an FQDN to host GitLab in a root path instead
# of using a relative URL.
# Documentation: http://doc.gitlab.com/ce/install/relative_url.html
# Uncomment and customize the following line to run in a non-root path
#
# ENV['RAILS_RELATIVE_URL_ROOT'] = "/gitlab"

worker_processes 2
working_directory "/srv/gitlab"
listen "0.0.0.0:8080", :tcp_nopush => true

# nuke workers after 30 seconds instead of 60 seconds (the default)
#
# NOTICE: git push over http depends on this value.
# If you want to be able to push huge amount of data to git repository over http
# you will have to increase this value too.
#
# Example of output if you try to push 1GB repo to GitLab over http.
#   -> git push http://gitlab.... master
#
#   error: RPC failed; result=18, HTTP code = 200
#   fatal: The remote end hung up unexpectedly
#   fatal: The remote end hung up unexpectedly
#
# For more information see http://stackoverflow.com/a/21682112/752049
#
timeout 60

# feel free to point this anywhere accessible on the filesystem
pid "/home/git/unicorn.pid"

# combine Ruby 2.0.0dev or REE with "preload_app true" for memory savings
# http://rubyenterpriseedition.com/faq.html#adapt_apps_for_cow
preload_app true

# Enable this flag to have unicorn test client connections by writing the
# beginning of the HTTP headers before calling the application.  This
# prevents calling the application for connections that have disconnected
# while queued.  This is only guaranteed to detect clients on the same
# host unicorn runs on, and unlikely to detect disconnects even on a
# fast LAN.
check_client_connection false

require_relative "/srv/gitlab/lib/gitlab/cluster/lifecycle_events"

before_exec do |server|
  # Signal application hooks that we're about to restart
  Gitlab::Cluster::LifecycleEvents.do_before_master_restart
end

run_once = true

before_fork do |server, worker|
  if run_once
    # There is a difference between Puma and Unicorn:
    # - Puma calls before_fork once when booting up master process
    # - Unicorn runs before_fork whenever new work is spawned
    # To unify this behavior we call before_fork only once (we use
    # this callback for deleting Prometheus files so for our purposes
    # it makes sense to align behavior with Puma)
    run_once = false

    # Signal application hooks that we're about to fork
    Gitlab::Cluster::LifecycleEvents.do_before_fork
  end

  # The following is only recommended for memory/DB-constrained
  # installations.  It is not needed if your system can house
  # twice as many worker_processes as you have configured.
  #
  # This allows a new master process to incrementally
  # phase out the old master process with SIGTTOU to avoid a
  # thundering herd (especially in the "preload_app false" case)
  # when doing a transparent upgrade.  The last worker spawned
  # will then kill off the old master process with a SIGQUIT.
  old_pid = "#{server.config[:pid]}.oldbin"
  if old_pid != server.pid
    begin
      sig = (worker.nr + 1) >= server.worker_processes ? :QUIT : :TTOU
      Process.kill(sig, File.read(old_pid).to_i)
    rescue Errno::ENOENT, Errno::ESRCH
    end
  end
  #
  # Throttle the master from forking too quickly by sleeping.  Due
  # to the implementation of standard Unix signal handlers, this
  # helps (but does not completely) prevent identical, repeated signals
  # from being lost when the receiving process is busy.
  # sleep 1
end

after_fork do |server, worker|
  # Signal application hooks of worker start
  Gitlab::Cluster::LifecycleEvents.do_worker_start

  # per-process listener ports for debugging/admin/migrations
  # addr = "127.0.0.1:#{9293 + worker.nr}"
  # server.listen(addr, :tries => -1, :delay => 5, :tcp_nopush => true)
end
