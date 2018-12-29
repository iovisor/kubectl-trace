#!/usr/bin/env bash
set -xeo pipefail

# Script that builds the bpftrace image and releases
# it to quay only if the request is coming from the main repo.

make=$(command -v make)
docker=$(command -v docker)

$make image/build

if [[ ! -z "$QUAY_TOKEN" ]]; then
  $docker login -u="fntlnz+travisci" -p="$QUAY_TOKEN" quay.io
  $make image/push

  if [[ "$TRAVIS_BRANCH" = "master" ]]; then
    $make image/latest
  fi
fi

