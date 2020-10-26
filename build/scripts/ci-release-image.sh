#!/usr/bin/env bash
set -xeo pipefail

# Script that builds and releases images to quay.io
# It is expected to be run from github actions.
# The secret "QUAY_TOKEN" is expected to be set, with access to a quay.io repository
# The environment variable "QUAY_BOT_USER" can be used to override the default bot username
QUAY_BOT_USER=${QUAY_BOT_USER:-kubectltrace_buildbot}

git_org=$(dirname $1)  # github.repository format: ORGNAME/REPONAME
git_ref=$2             # github.ref        format: refs/REMOTE/REF
                       #                       eg, refs/heads/BRANCH
                       #                           refs/tags/v0.9.6-pre
git_base_ref=${3}      # github.head_ref   if the build is a pull request, this will be the pull request branch name

git_ref_id=$(basename ${git_ref})

make=$(command -v make)
docker=$(command -v docker)

makeopts=""
if [[ ! -z "${git_org}" ]]; then
  makeopts="-e GIT_ORG=${git_org} ${makeopts}"
fi

if [[ ! -z "${git_base_ref}" ]]; then
  makeopts="-e GIT_BRANCH=${git_base_ref} ${makeopts}"
fi

if [[ ! -z "$QUAY_TOKEN" ]]; then
  $docker login -u="${git_org}+${QUAY_BOT_USER}" -p="$QUAY_TOKEN" quay.io
  $make $makeopts image/push

  if [[ "${git_ref_id}" = "master" ]] && [[ "${git_base_ref}" = "" ]]; then
    $make $makeopts image/latest
  fi
fi
