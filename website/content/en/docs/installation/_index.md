---
title: Installation
linkTitle: Installation
weight: 2
description: Install the Operator SDK CLI
---

- [Install from Homebrew (macOS)](#install-from-homebrew-macos)
- [Install from GitHub release](#install-from-github-release)
- [Compile and install from master](#compile-and-install-from-master)

## Install from Homebrew (macOS)

If you are using [Homebrew][homebrew_tool], you can install the SDK CLI tool with the following command:

```sh
brew install operator-sdk
```

## Install from GitHub release

#### Prerequisites

- [curl](https://curl.haxx.se/)
- [gpg](https://gnupg.org/) version 2.0+

#### 1. Download the release binary

Set platform information:

```sh
export ARCH=$(case $(arch) in x86_64) echo -n amd64 ;; aarch64) echo -n arm64 ;; *) echo -n $(arch) ;; esac)
export OS=$(uname | awk '{print tolower($0)}')
```

Download the binary for your platform:

```sh
export OPERATOR_SDK_DL_URL=https://github.com/operator-framework/operator-sdk/releases/latest/download
curl -LO ${OPERATOR_SDK_DL_URL}/operator-sdk_${OS}_${ARCH}
```

#### 2. Verify the downloaded binary

Import the operator-sdk release GPG key:

```sh
gpg --recv-keys 052996E2A20B5C7E
```

Download the checksums file and its signature, then verify the signature:

```sh
curl -LO ${OPERATOR_SDK_DL_URL}/checksums.txt
curl -LO ${OPERATOR_SDK_DL_URL}/checksums.txt.asc
gpg -u "Operator SDK (release) <cncf-operator-sdk@cncf.io>" --verify checksums.txt.asc
```

You should see something similar to the following:

```console
gpg: assuming signed data in 'checksums.txt'
gpg: Signature made Fri 30 Oct 2020 12:15:15 PM PDT
gpg:                using RSA key ADE83605E945FA5A1BD8639C59E5B47624962185
gpg: Good signature from "Operator SDK (release) <cncf-operator-sdk@cncf.io>" [ultimate]
```

Make sure the checksums match:

```sh
grep operator-sdk_${OS}_${ARCH} checksums.txt | sha256sum -c -
```

You should see something similar to the following:

```console
operator-sdk_linux_amd64: OK
```

#### 3. Install the release binary in your PATH

```sh
chmod +x operator-sdk_${OS}_${ARCH} && sudo mv operator-sdk_${OS}_${ARCH} /usr/local/bin/operator-sdk
```

## Compile and install from master

#### Prerequisites

- [git][git_tool]
- [mercurial][mercurial_tool] version 3.9+
- [bazaar][bazaar_tool] version 2.7.0+
- [go][go_tool] version v1.15+.

```sh
git clone https://github.com/operator-framework/operator-sdk
cd operator-sdk
git checkout master
make install
```

**Note:** Ensure that your `GOPROXY` is set with its default value for Go
versions 1.15+ which is `"https://proxy.golang.org|direct"`.

[homebrew_tool]:https://brew.sh/
[git_tool]:https://git-scm.com/downloads
[mercurial_tool]:https://www.mercurial-scm.org/downloads
[bazaar_tool]:http://wiki.bazaar.canonical.com/Download
[go_tool]:https://golang.org/dl/
