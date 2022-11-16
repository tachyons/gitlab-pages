#!/bin/sh
set -e

(>&2 echo "RHEL: Disable subscription manager, use only official UBI repos")
# Disable all repositories (to limit RHEL host repositories) and only use official UBI repositories
sed -i "s/enabled=1/enabled=0/" /etc/dnf/plugins/subscription-manager.conf
