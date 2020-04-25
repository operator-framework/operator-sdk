---
title: Operator Bundle Tooling in Operator SDK
linkTitle: CLI Overview
weight: 2
---

This document gives an overview of using `operator-sdk` to work with Operator [bundles][operator-bundles], namely on-disk metadata and manifests, and creating [bundles][operator-bundle-image].

## Commands

The following `operator-sdk` subcommands create or interact with Operator manifests, metadata, and bundles:

* `generate csv`: creates a new or updates an existing CSV in a `manifests/` directory and copies your CRDs to the versioned bundle directory. Read more about this command [here][sdk-generate-csv].
* `bundle create`: creates an Operator bundle from manifests and metadata on disk, or writes bundle metadata to disk. This subcommand has corresponding functionality to `opm alpha bundle build`. Output and underlying behavior between these commands is the same, except nothing is written to disk unless `--generate-only` is set (false by default). Refer to [`opm alpha bundle build` docs][operator-opm-build] for more information. CLI differences between these commands:
  {{<table "table table-striped table-bordered">}}
  | **operator-sdk** | **opm** |
  |-----|-----|
  | `operator-sdk bundle create --default=<channel-name>` |  `opm alpha bundle build --default-channel=<channel-name>` |
  | `operator-sdk bundle create --generate-only` | `opm alpha bundle generate` |
  | `operator-sdk bundle create <image-tag>` | `opm alpha bundle build --tag <image-tag>` |
  {{</table>}}
* `bundle validate`: validates an Operator bundle image or unpacked manifests and metadata. This subcommand has corresponding functionality to `opm alpha bundle validate`. Refer to the [`opm alpha bundle validate` docs][operator-opm-validate] for more information. CLI differences between these commands:
  {{<table "table table-striped table-bordered">}}
  | **operator-sdk** | **opm** |
  |-----|-----|
  | `operator-sdk bundle validate <image-tag>` | `opm alpha bundle validate --tag <image-tag>` |
  | `operator-sdk bundle validate <directory>` | no equivalent |
  {{</table>}}

[sdk-generate-csv]:/docs/olm-integration/generating-a-csv
[operator-bundles]:https://github.com/operator-framework/operator-registry/tree/v1.5.3#manifest-format
[operator-bundle-image]:https://github.com/operator-framework/operator-registry/blob/v1.5.3/docs/design/operator-bundle.md
[operator-opm-build]:https://github.com/operator-framework/operator-registry/blob/v1.5.9/docs/design/operator-bundle.md#build-bundle-image
[operator-opm-validate]:https://github.com/operator-framework/operator-registry/blob/v1.5.9/docs/design/operator-bundle.md#validate-bundle-image
