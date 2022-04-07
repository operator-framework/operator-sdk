---
title: Documentation
linkTitle: Documentation
weight: 20
---

If a contribution changes the user interface or existing APIs it must include new or updated documentation.
Since the operator-sdk repository does not expose many public packages, documentation mostly comes in the form of our website's [markdown docs][website-md].
Good [godocs][godocs] are expected nonetheless.

Likewise a [changelog fragment][changelog-template] should be added containing a summary of the change and optionally a migration guide.

## Testing docs changes

This document discusses how to visually inspect documentation changes as they would be applied
to the live website. All changes to documentation should be inspected locally before being pushed
to a PR.

### Prerequisites

The docs are built with [Hugo][hugo] which can be installed along with the
required extensions by following the [docsy install guide][docsy-install].

Note: Be sure to install hugo-extended.

We use `git submodules` to install the docsy theme. From the
`operator-sdk` directory, update the submodules to install the theme.

```sh
git submodule update --init --recursive
```

### Build and Serve

You can build and serve your docs to `localhost:1313`. From the `website/`
directory run:

```sh
hugo server
```

Any changes will be included in real time.

### Check Docs

`make test-docs` will validate changelog fragments, build doc HTML in a container, and check its links.
Please consider running this locally before creating a PR to save CI resources.

[hugo]:https://gohugo.io/
[docsy-install]:https://www.docsy.dev/docs/get-started/other-options/#prerequisites-and-installation
[website-md]:https://github.com/operator-framework/operator-sdk/tree/master/website/content/en/docs
[changelog-template]:https://github.com/operator-framework/operator-sdk/blob/master/changelog/fragments/00-template.yaml
[godocs]:https://blog.golang.org/godoc
