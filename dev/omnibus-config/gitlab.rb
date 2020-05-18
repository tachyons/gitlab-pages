postgresql['enable'] = true
redis['enable'] = true
unicorn['enable'] = false
puma['enable'] = false
sidekiq['enable'] = false
mailroom['enable'] = false
gitlab_exporter['enable'] = false

# gitaly
gitaly['enable'] = false
gitaly['socket_path'] = nil
gitaly['listen_addr'] = "0.0.0.0:8075"
gitaly['auth_token'] = 'gitalytokengitalytoken'
gitlab_rails['internal_api_url'] = 'http://webservice:8080'
gitlab_shell['secret_token'] = '6fad933c6267760415116fc4f35d2c7fc969f4ce0c162b49c3dd7be5517283e63000340ba7282dd97c2b3518b6d3c97a7cdd995dcb6f00dff11cf0aa316a459f'

gitlab_workhorse['enable'] = false
gitlab_workhorse['listen_addr'] = 'workhorse:8181'
gitlab_workhorse['listen_network'] = 'tcp'

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
gitlab_rails['ssh_host'] = "localhost"
gitlab_rails['ssh_host'] << ":#{ENV['GITLAB_SSH_PORT']}" if ENV['GITLAB_SSH_PORT']
gitlab_rails['secret_key_base'] = "678f063ed891fd6e431a1a22446ca5702fe7711730146f03169be2560b51b2176092a9fd6121f185a32e429ffea64a738af5ec56da74828a83eee101c081049b"
gitlab_rails['db_key_base'] = "76fd4c76ad88731035c849ba42fbe298742776b52a2c27c25181a08a8d4b1755ae19870fec0bca8b21060a2f1b960ff958ba861e25580a1474e1f3bb63152883"
gitlab_rails['otp_key_base'] = "e62021080e20756e9a650079ac9794a88f0d312331512d639e468d8aaf8bd44b45322884197f3cd5bd5c699fe4949aee29bb8fa9d749260cac37f7294c04d1de"
gitlab_rails['openid_connect_signing_key'] = <<EOS
-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEA8j0zUt2skIwEipO0lQC+SdeOxYTE/Lnkszx2b49wpD7pyrAf
Si7WI5Z4dTanFaBoA2z79TWZ2kKP7aS5AKOeOmVCAR+kdE+e40bjRncagnhRZMP7
A4GYQu43h/WY6xQNGQ3b/W1wfZ6TKoHqrDtw53ykPf9CX/bKehDXgMlG/rjkmiJY
RZ09wU7of9971uhKvp3rMFiIfMjU2jAiQi0rz0NTvQNNFi+bmXTBB+th/l6x7dR+
iwndjOyMPtag+pW1q/zStVPSXGC3KrDerZIWgLBo/kIf64CvDAG6bzNT5C+UAvtD
uBX3lu6g7YOry8H2YskE/rdxbu9O82umadDSNQIDAQABAoIBAQDmVlVwLUMCa3GC
B7NFn+BRPLwQiw/4tBaIbCy+JenY883BIrmOA30xd8nwwtgb01w0LbRjVOWHyZUs
Rl54AhzZsCEUCm+I/ZC60lQYU/aMKzZSF5yel+MdeYCqMlmG5pkRwsD1eNPnTxrm
54Qq45aXN0jBfvAxaezcJxmT51AvsexsCu/mjBH7rabQMLRWK1eHhejzObk6U7HM
mp0IkgzdPpTnlQydzwZd2+CVHJAxgIgDJnztnnZK2CGI07j2EjxUB+VjvJPiMkcU
WYSsFqPrWBz1baVzUtGxxcbMxjdJqkX53i8d1Ja87bnzTSEFgZ6gYVQoHrfsw0C4
Y0/IFSXdAoGBAPr66rmcApMGGFsY51UcgzOZ+cb81i5gSO32gy6yxDrgSsTjo3hY
2j95UW7FawlYTxHGMVgIOTHJaZf37nPLVXUXqXxXcYi0+jFNZ8tfH0Q7LdAI38PR
EyUb3uIOi9ZpiptehJ4527C/+zmDkqknH7zPkDd0G70xbWP0wGch5BLDAoGBAPcV
hunFjzYoqBV7NzcvdWV6zAt7jp/6/7wqJanHfP+1dwTY8oHhmLKsiG8846HInFX2
BtXlBSF+k7vj7mBa4rotCmKhdg0KdwUFj+0ucyZlUpTkulArZrHcjRv+UgB32UFv
pVi57krTkFoHnpwz4+yOyh0thmEM+5RPBInhkcenAn9Glktt6HLQ5uXyuQ5W4a4E
1UlqrbGPWYUpaCt89so5xNtChMN1FYhMUdKVVPDFsBCM7zDxkV6EoiEQnKpz5rwi
MoV4cx97U2sKcrq+eAJTLlG0MMq+SA8aYLxO18h8hTJ2gB7MYXloPJZrlpWo/Vfv
wnDNJ9qlW0Fo221BlEaRAoGBALKeRE6g+/up/oCw92SUkWiQEgvHAX8Tg2ZyjZGj
3fA5LdY6Q22zOaEI6wSHXwOos6PM3HoqOsBQ8dmlV8EeUsNGJjjRAjdBNnNR5w8O
Qj4ih0WN6EZmabHwf64Z9wvRmCQe4sg2eVZshPEgiv/e30k2nFPlgjyMlx5K1Twb
7jgHAoGAG9RR57lewMcL3Fki9IfYNfi+d3oeHK3fvrv+bku/jcyIGN0Ao61f3HBS
8qldV1eBBGOiezNLTeixok054kLnAe9l6pw4ArLouShaAo0+XgTYtUMkGO+sgCR/
4OzsVuwTfkQ8oB9p0nGqFvkOGQF9Kejg6a/aNWGrFiSy92gtQ0s=
-----END RSA PRIVATE KEY-----
EOS
