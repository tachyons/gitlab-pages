#!/bin/sh
set -e

(>&2 echo "Remediating: '	xccdf_org.ssgproject.content_rule_use_pam_wheel_for_su'")

if [ -e "/etc/pam.d/su"]; then
    sed '/^[[:space:]]*#[[:space:]]*auth[[:space:]]\+required[[:space:]]\+pam_wheel\.so[[:space:]]\+use_uid$/s/^[[:space:]]*#//' \
      -i /etc/pam.d/su
fi
