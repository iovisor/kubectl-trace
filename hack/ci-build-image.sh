#!/usr/bin/env bash
set -xeo pipefail

make=$(command -v make)

makeopts=""
if [[ ! -z "$TRAVIS_PULL_REQUEST_BRANCH" ]]; then
  makeopts="-e GIT_BRANCH=$TRAVIS_PULL_REQUEST_BRANCH image/build"
fi

$make $makeopts image/build
