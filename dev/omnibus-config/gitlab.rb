postgresql['enable'] = true
redis['enable'] = false
redis_exporter['enable'] = false
unicorn['enable'] = false
puma['enable'] = false
sidekiq['enable'] = false
mailroom['enable'] = false
gitlab_exporter['enable'] = false
nginx['enable'] = false
gitaly['enable'] = false
gitlab_workhorse['enable'] = false

# PostgreSQL configuration
postgresql['listen_address'] = '0.0.0.0'
postgresql['trust_auth_cidr_addresses'] = %w(127.0.0.0/24)
postgresql['md5_auth_cidr_addresses'] = %w(0.0.0.0/0)
postgresql['sql_user_password'] = '791e36bc4780b2402bc6f29f082dfc52'

postgres_exporter['env'] ={
  'DATA_SOURCE_NAME' => "user=gitlab-psql host=0.0.0.0 database=postgres"
}

gitlab_rails['auto_migrate'] = false
