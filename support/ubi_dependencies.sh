#!/bin/bash

package_dependencies=()
external_dependencies=()

# Update DNF / RPM package data
DNF=dnf
if [ -f /usr/bin/microdnf ]; then
  DNF=microdnf
fi
${DNF} update

# List out all external dependencies to Rails gems (under vendor)
external_files_needed=$(find /usr/lib64/ruby/gems /srv/ /usr/local/bin/ /usr/local/lib/ -type f -executable | xargs ldd | awk 'match($0, /=> (.*) \(0x/, m){ print m[1]}' | sort | uniq -c)

# Find all packages that provide these files
echo
echo "## Walking linked files"
for filename in $external_files_needed ; do
  if [[ $filename =~ ^[0-9]+$ ]] ; then
    continue 
  fi

  provided_by=$(rpm -q --whatprovides $filename)
  if [[ "${provided_by}" =~ 'not owned' || "${provided_by}" =~ 'no package provides' ]]; then
    echo $provided_by
    external_dependencies+=($filename)
    continue
  fi

  echo "$provided_by provides $filename"
  package_dependencies+=($provided_by)
done

echo
echo "## PACKAGES NEEDED"
for f in "${package_dependencies[@]}"; do echo "${f}"; done | sort -u

echo
echo "## EXTERNAL DEPENDENCIES"
for f in "${external_dependencies[@]}"; do echo "${f}"; done | sort -u
