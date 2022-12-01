#!/bin/bash

patchdir=$1
ruby_version=$2

# Verify patches
while read -r patchname; do
    patchfile="${patchdir}/${ruby_version}/${patchname}.patch"
    if [[ ! -f "${patchfile}" ]]; then
    echo "!! Missing mandatory patch ${patchname}"
    echo "!! Make sure ${patchfile} exists before proceeding."
    exit 1
    fi
done < "${patchdir}/mandatory_patches"

# Apply patches
if [[ -d "${patchdir}/${ruby_version}" ]]; then
    for i in "${patchdir}/${ruby_version}"/*.patch; do
    echo "$i..."
    patch -p1 -i "$i"
    done
fi
