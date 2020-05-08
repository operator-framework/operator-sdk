---
title: Build and serve the docs locally
linkTitle: Local Docs
---

## Prerequisites

Clone the repository:

```
$ git clone https://github.com/operator-framework/operator-sdk/
```

The docs are built with [Hugo](https://gohugo.io/) which can be installed along with the
required extensions by following the [docsy install
guide](https://www.docsy.dev/docs/getting-started/#prerequisites-and-installation).

Note: Be sure to install hugo-extended.

We use `git submodules` to install the docsy theme. From the
`operator-sdk` directory, update the submodules to install the theme.

```
$ git submodule update --init --recursive
```

## Build and Serve

You can build and serve your docs to localhost:1313. From the `website`
directory run:

```
hugo server
```

Any changes will be included in real time.


## Check Links

`make test-links` will use containers to build html and check the links.
Please consider running this locally before creating a PR to save CI resources.
