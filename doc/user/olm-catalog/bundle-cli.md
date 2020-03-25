# Operator Bundle Tooling in `operator-sdk`

This document gives an overview of using `operator-sdk` to work with Operator [bundles][registry-bundle], namely on-disk bundle directories and creating bundle [images/metadata][registry-bundle-image].

## Commands

The following `operator-sdk` subcommands create or interact with Operator on-disk bundles and bundle images:

* `generate csv`: creates a new or updates an existing CSV in a semantically versioned bundle directory, creates a package manifest if it does not exist, and optionally copies your CRDs to the versioned bundle directory. Read more about this command [here][sdk-generate-csv].
* `bundle create`: creates an Operator bundle image from manifests on disk, or writes bundle image metadata to disk. This subcommand has corresponding functionality to `opm alpha bundle build`, and is stable. Output and underlying behavior between these commands is the same, except nothing is written to disk unless `--generate-only` is set (unset by default). Refer to [`opm alpha bundle build` docs][registry-opm-build] for more information. CLI differences between these commands:
  | **operator-sdk** | **opm** |
  |--- |--- |
  | `operator-sdk bundle create --default=<channel-name>` |  `opm alpha bundle build --default-channel=<channel-name>` |
  | `operator-sdk bundle create --generate-only` | `opm alpha bundle generate` |
  | `operator-sdk bundle create <image-tag>` | `opm alpha bundle build --tag <image-tag>` |
  | no equivalent | `opm alpha bundle build --overwrite` |
  | `operator-sdk bundle validate <image-tag>` | `opm alpha bundle validate --tag <image-tag>` |
* `bundle validate`: validates an Operator bundle image. This subcommand has corresponding functionality to `opm alpha bundle validate`, and is stable. Refer to the [`opm alpha bundle validate` docs][registry-opm-validate] for more information. CLI differences between these commands:
  | **operator-sdk** | **opm** |
  |--- |--- |
  | `operator-sdk bundle validate <image-tag>` | `opm alpha bundle validate --tag <image-tag>` |

[sdk-generate-csv]:./generating-a-csv.md
[registry-bundle]:https://github.com/operator-framework/operator-registry/tree/v1.5.3#manifest-format
[registry-bundle-image]:https://github.com/operator-framework/operator-registry/blob/v1.5.3/docs/design/operator-bundle.md
[registry-opm-build]:https://github.com/operator-framework/operator-registry/blob/v1.5.9/docs/design/operator-bundle.md#build-bundle-image
[registry-opm-validate]:https://github.com/operator-framework/operator-registry/blob/v1.5.9/docs/design/operator-bundle.md#validate-bundle-image
