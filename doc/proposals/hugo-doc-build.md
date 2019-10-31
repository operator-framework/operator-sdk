---
title: Build and Host Documentation
authors:
  - "@asmacdo"
reviewers:
  - TBD
approvers:
  - TBD
creation-date: 2019-10-29
last-updated: 2019-10-29
status: TBD
---

# Build and Host Documentation

## Release Signoff Checklist

- \[ \] Enhancement is `implementable`
- \[ \] Design details are appropriately documented from clear requirements
- \[ \] Test plan is defined
- \[ \] Graduation criteria for dev preview, tech preview, GA
- \[ \] User-facing documentation is created in [operator-sdk/doc][operator-sdk-doc]

## Motivation

Hosted documentation will improve the Operator SDK user experience by
increasing the visibility and accessibility of the content. 

## Summary

This enhancement proposes a technology stack that will be used to write,
build, and host the documentation for the Operator SDK project.

Hugo will be used for static site generation from Markdown. Neflify will
be used to host the site. 

Highlights:
  - Files will remain markdown, few syntax changes necessary.
  - Stack is aligned with current Golang/Kubernetes community.
  - Proof of concept: https://operator-sdk.netlify.com/

## Open Questions (optional)

- What domain should we use? Available options include operator-sdk.com,
  operator-sdk.org, operator-framework.com, and operator-framework.org

### Goals

1. Implementation of this proposal will result in a website that hosts
   the documentation that currently exists in the `doc` directory of
   [the Operator SDK
   project](https://github.com/operator-framework/operator-sdk/). 
1. A minimal landing page will be created.
1. Docs will build automatically when code merges to master branch.

### Non-Goals

1. Content changes. Other than minor syntax changes for Hugo compliance,
   the content should remain the same, including the content
   organization.

## Proposal

Docs will be built with [Hugo](https://gohugo.io/). Highlights:
 - Static site generator
 - Files will remain markdown, with minor syntactical changes.
 - Fast builds
 - Commonly used for Golang projects, including Docker and Kubernetes. 
 - Open source (Apache 2.0)
 - Multi-language support

Hugo can use themes to lower the barrier to entry.
[Docsy](https://github.com/google/docsy) is a theme specifically
designed for medium to large technical documentation sets, and supports
multiple content types (tutorials, reference docs, blog posts, and so
on). Docsy is used by other projects in the Kubernetes community,
including Knative and Kubeflow.
[kubernetes.io](https://github.com/kubernetes/website) is planning to
switch to docsy as well.

Key features:
 - Auto-generated Navigation
 - Versioned documentation support
 - Search
 - GitHub integration for fixing or reporting issues.
 - Open source (Apache 2.0)

Docs will be deployed with [Netlify](https://www.netlify.com/).

Key Features:
  - Free hosting with [Open Source Project
      Plan](https://www.netlify.com/legal/open-source-policy/)
  - Open source plan is equivalent to [Pro
      plan](https://www.netlify.com/pricing/#teams)
  - Simple GitHub integration
  - "Deploy Preview" feature builds documentation for pull requests.

### Risks and Mitigations

If the Docsy theme (or similar) does not work for our needs, customizing
Hugo templates can be time consuming to create and maintain. However,
our documentation needs are not unique, we should be able to get most or
all of what we need out of the box.

## Drawbacks

Website maintenance can be time consuming, even if everything goes
well. This stack seems like a relatively efficient way to host docs, but
there are even lighter weight solutions.

## Alternatives

Because our documentation needs are not unique, there are many tool sets
that could meet our needs. A similar arrangement would be `.rst` files,
rendered with `sphinx` and hosted with readthedocs, but that would
require a more significant refactor.

## Infrastructure Needed (optional)

1. (optional) Domain registration
1. SSL Cert via Let's Encrypt
1. Installation of Netlify app for GitHub repository.

[operator-sdk-doc]:  ../../doc
