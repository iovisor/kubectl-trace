#!/bin/bash

set -eu

# Taken from
# https://gohugo.io/hosting-and-deployment/hosting-on-github/#build-and-deployment

cd docs
hugo
cd public
git add --all
git commit -m "Publishing to gh-pages"
cd ../..
