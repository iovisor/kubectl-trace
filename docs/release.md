# kubectl-trace Release Process

Our release process is automated using [goreleaser](https://github.com/goreleaser/goreleaser)
and automatically publishes the release to [krew](https://github.com/kubernetes-sigs/krew-index)

When we release we do the following process:

1. We decide together (usually in the #kubectl-trace channel in Kubernetes slack) what's the next version to tag
2. A maintainer will tag master at a passing CI revision
3. Pushing the tag will result in goreleaser github action creating a release, and publishing this to the `krew` index

## Release commands

Tag the version:

```bash
VERSION=v0.1.2
git tag -a $VERSION -m "$VERSION"
git push origin $VERSION
```

From there, github actions should automatically create the release, as it sets
the `GITHUB_TOKEN`.

### Manual release

In case you need to run a release manually:

```
export GITHUB_TOKEN=<YOUR_GH_TOKEN>
make release
```

Though this will not automatically update the krew index, and will require
administrative privileges on the repo.
