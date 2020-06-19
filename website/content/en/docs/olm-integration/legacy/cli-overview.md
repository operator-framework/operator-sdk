---
title: OLM and Bundle CLI Overview
linkTitle: CLI Overview
weight: 10
---

This document gives an overview of using `operator-sdk` to work with Operator manifests related to OLM,
namely [bundles][bundle] and [package manifests][package-manifests]. See the [manifests generation][doc-olm-generate]
doc for an in-depth discussion of these commands.

## Commands

### OLM installation

The following `operator-sdk` subcommands manage an OLM installation:

- [`olm install`][cli-olm-install]: install a particular version of OLM.
- [`olm status`][cli-olm-status]: check the status of a particular version of OLM running in a cluster. This command
can infer the version of an error-free OLM installation.
- [`olm uninstall`][cli-olm-uninstall]: uninstall a particular version of OLM running in a cluster. This command
can infer the version of an error-free OLM installation.

### Manifests and metadata

The following `operator-sdk` subcommands create or interact with Operator package manifests and bundles:

##### Bundles

- [`generate bundle`][cli-gen-bundle]: creates a new or updates an existing bundle in the
`deploy/olm-catalog/<operator-name>` directory. This command handles generating both manifests and metadata.
- [`bundle validate`][cli-bundle-validate]: validates an Operator bundle image or unpacked manifests and metadata.

##### Package Manifests

- [`generate packagemanifests`][cli-gen-packagemanifests]: creates a new or updates an existing versioned
directory as part of the package manifests in the `deploy/olm-catalog/<operator-name>` directory.
- [`run packagemanifests`][doc-testing-deployment]: runs an Operator's package manifests format
with an existing OLM installation.


[bundle]:https://github.com/operator-framework/operator-registry/blob/v1.12.6/docs/design/operator-bundle.md
[package-manifests]:https://github.com/operator-framework/operator-registry/tree/v1.5.3#manifest-format
[doc-olm-generate]:/docs/olm-integration/legacy/generating-a-csv
[cli-olm-install]:/docs/cli/operator-sdk_olm_install
[cli-olm-status]:/docs/cli/operator-sdk_olm_status
[cli-olm-uninstall]:/docs/cli/operator-sdk_olm_uninstall
[cli-gen-bundle]:/docs/cli/operator-sdk_generate_bundle
[cli-gen-packagemanifests]:/docs/cli/operator-sdk_generate_packagemanifests
[cli-bundle-validate]:/docs/cli/operator-sdk_bundle_validate
[doc-testing-deployment]:/docs/olm-integration/legacy/testing-deployment
