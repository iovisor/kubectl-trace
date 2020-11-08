---
title: Installing
weight: -20
---

There are a couple ways to install `kubectl-trace` on your machine.

## Krew

You can install `kubectl trace` using the [Krew](https://github.com/kubernetes-sigs/krew), the package manager for kubectl plugins.

Once you have [Krew installed](https://krew.sigs.k8s.io/docs/user-guide/setup/install/) just run:

```bash
kubectl krew install trace
```

You're ready to go!

## Pre-built binaries

See the [release](https://github.com/iovisor/kubectl-trace/releases) page for the full list of pre-built assets.

The commands here show `amd64` versions, `386` versions are available in the releases page.

**Linux**

```bash
curl -L -o kubectl-trace.tar.gz https://github.com/iovisor/kubectl-trace/releases/download/v0.1.0-rc.1/kubectl-trace_0.1.0-rc.1_linux_amd64.tar.gz
tar -xvf kubectl-trace.tar.gz
mv kubectl-trace /usr/local/bin/kubectl-trace
```

**OSX**

```bash
curl -L -o kubectl-trace.tar.gz https://github.com/iovisor/kubectl-trace/releases/download/v0.1.0-rc.1/kubectl-trace_0.1.0-rc.1_darwin_amd64.tar.gz
tar -xvf kubectl-trace.tar.gz
mv kubectl-trace /usr/local/bin/kubectl-trace
```


**Windows**

In PowerShell v5+
```powershell
$url = "https://github.com/iovisor/kubectl-trace/releases/download/v0.1.0-rc.1/kubectl-trace_0.1.0-rc.1_windows_amd64.zip"
$output = "$PSScriptRoot\kubectl-trace.zip"

Invoke-WebRequest -Uri $url -OutFile $output
Expand-Archive "$PSScriptRoot\kubectl-trace.zip" -DestinationPath "$PSScriptRoot\kubectl-trace"
```

## Source

```
go get -u github.com/iovisor/kubectl-trace/cmd/kubectl-trace
```

This will download and compile `kubectl-trace` so that you can use it as a kubectl plugin with `kubectl trace`

## Packages

You can't find the package for your distro of choice?
You are very welcome and encouraged to create it and then [open an issue](https://github.com/iovisor/kubectl-trace/issues/new) to inform us for review.

### Arch - AUR

The official [PKGBUILD](https://aur.archlinux.org/cgit/aur.git/tree/PKGBUILD?h=kubectl-trace-git) is on AUR.

If you use `yay` to manage AUR packages you can do:

```
yay -S kubectl-trace-git
```
