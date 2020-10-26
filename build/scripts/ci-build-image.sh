#!/usr/bin/env bash
set -xeo pipefail

git_ref=$1     # github.ref        format: refs/REMOTE/REF
               #                       eg, refs/heads/BRANCH
               #                           refs/tags/v0.9.6-pre

git_base_ref=$(basename ${git_ref})

make=$(command -v make)

if [[ ! -z "${git_base_ref}" ]]; then
  makeopts="-e GIT_BRANCH=${git_base_ref} ${makeopts}"
fi

$make $makeopts image/build
$make $makeopts image/build-init
