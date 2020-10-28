#!/usr/bin/env bash
set -xeo pipefail

# Script that builds and releases images to quay.io
# It is expected to be run from github actions.
# The secret "QUAY_TOKEN" is expected to be set, with access to a quay.io repository
# The environment variable "QUAY_BOT_USER" can be used to override the default bot username
GIT_ORG=${GIT_ORG:-iovisor}
QUAY_BOT_USER=${QUAY_BOT_USER:-kubectltrace_buildbot}

git_ref=$1             # github.ref        format: refs/REMOTE/REF
                       #                       eg, refs/heads/BRANCH
                       #                           refs/tags/v0.9.6-pre
git_base_ref=$(basename "${git_ref}")

make=$(command -v make)
docker=$(command -v docker)

makeopts=""
if [[ -n "${git_base_ref}" ]]; then
  makeopts="-e GIT_BRANCH=${git_base_ref} ${makeopts}"
fi

if [[ -n "$QUAY_TOKEN" ]]; then
  $docker login -u="${GIT_ORG}+${QUAY_BOT_USER}" -p="$QUAY_TOKEN" quay.io
  $make "$makeopts" image/push

  if [[ "${git_base_ref}" = "master" ]]; then
    $make "$makeopts" image/latest
  fi
fi
