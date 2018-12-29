#!/usr/bin/env bash
set -xeo pipefail

# Script that builds the bpftrace image and releases
# it to quay only if the request is coming from the main repo.

make=$(command -v make)
docker=$(command -v docker)

makeopts=""
if [[ ! -z "$TRAVIS_PULL_REQUEST_BRANCH" ]]; then
  makeopts="-e GIT_BRANCH=$TRAVIS_PULL_REQUEST_BRANCH image/build"
fi

$make $makeopts image/build

if [[ ! -z "$QUAY_TOKEN" ]]; then
  $docker login -u="fntlnz+travisci" -p="$QUAY_TOKEN" quay.io
  $make $makeopts image/push
fi

if [[ "$TRAVIS_BRANCH" = "master" && "$TRAVIS_PULL_REQUEST_BRANCH" = "" ]]; then
  $make $makeopts image/latest
fi
