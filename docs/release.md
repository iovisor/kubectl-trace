# kubectl-trace Release Process

Our release process is automated using [goreleaser](https://github.com/goreleaser/goreleaser).

When we release we do the following process:

- We decide together (usually in the #kubectl-trace channel in Kubernetes slack) what's the next version to tag
- A person with repository rights does the tag
- The same person runs goreleaser in their machine
- The tag is live on Github with the artifacts
- Travis will build the tag and push the related docker images

## Release commands

Tag the version

```bash
git tag -a v0.1.0-rc.0 -m "v0.1.0-rc.0"
git push origin v0.1.0-rc.0
```

Run goreleaser, make sure to export your GitHub token first.

```
export GITHUB_TOKEN=`YOUR_GH_TOKEN`
make cross
```

