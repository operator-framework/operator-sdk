---
title: Documentation
linkTitle: Documentation
weight: 20
---

This document discusses how to visually inspect documentation changes as they would be applied
to the live website. All changes to documentation should be inspected locally before being pushed
to a PR.

## Prerequisites

The docs are built with [Hugo][hugo] which can be installed along with the
required extensions by following the [docsy install guide][docsy-install].

Note: Be sure to install hugo-extended.

We use `git submodules` to install the docsy theme. From the
`operator-sdk` directory, update the submodules to install the theme.

```sh
git submodule update --init --recursive
```

## Build and Serve

You can build and serve your docs to `localhost:1313`. From the `website/`
directory run:

```sh
hugo server
```

Any changes will be included in real time.


## Check Links

`make test-links` will use containers to build html and check the links.
Please consider running this locally before creating a PR to save CI resources.


[hugo]:https://gohugo.io/
[docsy-install]:https://www.docsy.dev/docs/getting-started/#prerequisites-and-installation
