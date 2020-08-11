---
title: "Installation"
linkTitle: "Installation"
date: 2020-03-25
weight: 2
description: Install the Operator SDK CLI
---

- [Prerequisites](#prerequisites)
- [Install from Homebrew (macOS)](#install-from-homebrew-macos)
- [Install from GitHub release](#install-from-github-release)
- [Compile and install from master](#compile-and-install-from-master)

## Prerequisites

- [docker][docker-tool] version 17.03+ (or another tool compatible with multi-stage Dockerfiles).
- [kubectl][kubectl-tool] version v1.11.3+ (v1.16.0+ if using `apiextensions.k8s.io/v1` CRDs).

[docker-tool]:https://docs.docker.com/install/
[kubectl-tool]:https://kubernetes.io/docs/tasks/tools/install-kubectl/

## Install from Homebrew (macOS)

If you are using [Homebrew][homebrew_tool], you can install the SDK CLI tool with the following command:

```sh
$ brew install operator-sdk
```

## Install from GitHub release

### Download the release binaries

```sh
# Set the release version variable
$ RELEASE_VERSION=v1.0.0
# Linux
$ curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu
$ curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/ansible-operator-${RELEASE_VERSION}-x86_64-linux-gnu
$ curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/helm-operator-${RELEASE_VERSION}-x86_64-linux-gnu
# macOS
$ curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/operator-sdk-${RELEASE_VERSION}-x86_64-apple-darwin
$ curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/ansible-operator-${RELEASE_VERSION}-x86_64-apple-darwin
$ curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/helm-operator-${RELEASE_VERSION}-x86_64-apple-darwin
```

#### Verify the downloaded release binaries

```sh
# Linux
$ curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu.asc
$ curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/ansible-operator-${RELEASE_VERSION}-x86_64-linux-gnu.asc
$ curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/helm-operator-${RELEASE_VERSION}-x86_64-linux-gnu.asc
# macOS
$ curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/operator-sdk-${RELEASE_VERSION}-x86_64-apple-darwin.asc
$ curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/ansible-operator-${RELEASE_VERSION}-x86_64-apple-darwin.asc
$ curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/helm-operator-${RELEASE_VERSION}-x86_64-apple-darwin.asc
```

To verify a release binary using the provided asc files, place the binary and corresponding asc file into the same directory and use the corresponding command:

```sh
# Linux
$ gpg --verify operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu.asc
$ gpg --verify ansible-operator-${RELEASE_VERSION}-x86_64-linux-gnu.asc
$ gpg --verify helm-operator-${RELEASE_VERSION}-x86_64-linux-gnu.asc
# macOS
$ gpg --verify operator-sdk-${RELEASE_VERSION}-x86_64-apple-darwin.asc
$ gpg --verify ansible-operator-${RELEASE_VERSION}-x86_64-apple-darwin.asc
$ gpg --verify helm-operator-${RELEASE_VERSION}-x86_64-apple-darwin.asc
```

If you do not have the maintainers public key on your machine, you will get an error message similar to this:

```sh
$ gpg --verify operator-sdk-${RELEASE_VERSION}-x86_64-apple-darwin.asc
$ gpg: assuming signed data in 'operator-sdk-${RELEASE_VERSION}-x86_64-apple-darwin'
$ gpg: Signature made Fri Apr  5 20:03:22 2019 CEST
$ gpg:                using RSA key <KEY_ID>
$ gpg: Can't check signature: No public key
```

To download the key, use the following command, replacing `$KEY_ID` with the RSA key string provided in the output of the previous command:

```sh
$ gpg --recv-key "$KEY_ID"
```

You'll need to specify a key server if one hasn't been configured. For example:

```sh
$ gpg --keyserver keyserver.ubuntu.com --recv-key "$KEY_ID"
```

Now you should be able to verify the binary.

### Install the release binary in your PATH

```sh
# Linux
$ chmod +x operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu && sudo mkdir -p /usr/local/bin/ && sudo cp operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu /usr/local/bin/operator-sdk && rm operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu
$ chmod +x ansible-operator-${RELEASE_VERSION}-x86_64-linux-gnu && sudo mkdir -p /usr/local/bin/ && sudo cp ansible-operator-${RELEASE_VERSION}-x86_64-linux-gnu /usr/local/bin/ansible-operator && rm ansible-operator-${RELEASE_VERSION}-x86_64-linux-gnu
$ chmod +x helm-operator-${RELEASE_VERSION}-x86_64-linux-gnu && sudo mkdir -p /usr/local/bin/ && sudo cp helm-operator-${RELEASE_VERSION}-x86_64-linux-gnu /usr/local/bin/helm-operator && rm helm-operator-${RELEASE_VERSION}-x86_64-linux-gnu
# macOS
$ chmod +x operator-sdk-${RELEASE_VERSION}-x86_64-apple-darwin && sudo mkdir -p /usr/local/bin/ && sudo cp operator-sdk-${RELEASE_VERSION}-x86_64-apple-darwin /usr/local/bin/operator-sdk && rm operator-sdk-${RELEASE_VERSION}-x86_64-apple-darwin
$ chmod +x ansible-operator-${RELEASE_VERSION}-x86_64-apple-darwin && sudo mkdir -p /usr/local/bin/ && sudo cp ansible-operator-${RELEASE_VERSION}-x86_64-apple-darwin /usr/local/bin/ansible-operator && rm ansible-operator-${RELEASE_VERSION}-x86_64-apple-darwin
$ chmod +x helm-operator-${RELEASE_VERSION}-x86_64-apple-darwin && sudo mkdir -p /usr/local/bin/ && sudo cp helm-operator-${RELEASE_VERSION}-x86_64-apple-darwin /usr/local/bin/helm-operator && rm helm-operator-${RELEASE_VERSION}-x86_64-apple-darwin
```

## Compile and install from master

### Prerequisites for compilation

- [git][git_tool]
- [mercurial][mercurial_tool] version 3.9+
- [bazaar][bazaar_tool] version 2.7.0+
- [go][go_tool] version v1.13+.

```sh
$ git clone https://github.com/operator-framework/operator-sdk
$ cd operator-sdk
$ git checkout master
$ make tidy
$ make install
```

**Note:** Ensure that your `GOPROXY` is set with its default value for Go
versions 1.13+ which is `https://proxy.golang.org,direct`.

[homebrew_tool]:https://brew.sh/
[git_tool]:https://git-scm.com/downloads
[mercurial_tool]:https://www.mercurial-scm.org/downloads
[bazaar_tool]:http://wiki.bazaar.canonical.com/Download
[go_tool]:https://golang.org/dl/
