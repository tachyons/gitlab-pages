postgresql['enable'] = true
redis['enable'] = true
unicorn['enable'] = false
puma['enable'] = false
sidekiq['enable'] = false
mailroom['enable'] = false
gitlab_exporter['enable'] = false
nginx['enable'] = false
gitaly['enable'] = false

# PostgreSQL configuration
postgresql['listen_address'] = '0.0.0.0'
postgresql['trust_auth_cidr_addresses'] = %w(127.0.0.0/24)
postgresql['md5_auth_cidr_addresses'] = %w(0.0.0.0/0)
postgresql['sql_user_password'] = '791e36bc4780b2402bc6f29f082dfc52'

postgres_exporter['env'] ={
  'DATA_SOURCE_NAME' => "user=gitlab-psql host=0.0.0.0 database=postgres"
}

# Redis configuration
redis['bind'] = '0.0.0.0'
redis['port'] = 6379
redis['password'] = 'redis-meercat'
gitlab_rails['redis_host'] = 'omnibus'
gitlab_rails['redis_port'] = 6379
redis_exporter['flags'] = {
  'redis.addr' => 'redis://omnibus:6379',
  'redis.password' => 'redis-meercat'
}
gitlab_rails['redis_password'] = 'redis-meercat'
gitlab_rails['redis_socket'] = nil
gitlab_rails['auto_migrate'] = false
