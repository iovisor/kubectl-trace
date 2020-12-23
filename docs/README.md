# kubectl-trace docs

This directory contains the [Hugo][hugo] assets to generate the kubectl-trace [website][website].

## Publishing instructions

*These instructions are based off the ones on the official Hugo website [here][hugoGHPages].*

1. Install Hugo.
2. Preview the website by changing the working directory to the `docs/` directory and running
   `hugo server -D`.
3. Configure the `docs/public` folder (where Hugo will generate the static files) with git's
   worktree feature: `git worktree add -B gh-pages public upstream/gh-pages`.
3. If all looks good, from the root of the repository, run the `docs/bin/commit-gh-pages-files.sh`
    script. This will build all the static assets needed for GitHub pages in the `docs/public` folder
    which is `.gitignore`'ed on the `master` branch and instead serves as the worktree for the
    `gh-pages` branch.
4. If all looks good, push the `gh-pages`.

[hugo]: https://gohugo.io/
[hugoGHPages]: https://gohugo.io/hosting-and-deployment/hosting-on-github/
[website]: https://iovisor.github.io/kubectl-trace/
