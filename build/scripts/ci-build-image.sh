#!/usr/bin/env bash
set -xeo pipefail

git_org=$(dirname $1)  # github.repository format: ORGNAME/REPONAME
git_base_ref=${2}      # github.head_ref   if the build is a pull request, this will be the pull request branch name

make=$(command -v make)

makeopts=""
if [[ ! -z "${git_org}" ]]; then
  makeopts="-e GIT_ORG=${git_org} ${makeopts}"
fi

if [[ ! -z "${git_base_ref}" ]]; then
  makeopts="-e GIT_BRANCH=${git_base_ref} ${makeopts}"
fi

$make $makeopts image/build
$make $makeopts image/build-init
