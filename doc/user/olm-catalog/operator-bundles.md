# Operator Bundles

This document gives an overview of using `operator-sdk` to work with Operator [bundles][registry-bundle], namely on-disk bundle directories and creating bundle [images/metadata][registry-bundle-image].

## Commands

The following `operator-sdk` subcommands create or interact with Operator on-disk bundles and bundle images:

* `generate csv`: creates a new or updates an existing CSV in a semantically versioned bundle directory, creates a package manifest if it does not exist, and optionally copies your CRDs to the versioned bundle directory. Read more about this command [here][sdk-generate-csv].
* `bundle create`: creates an Operator bundle image from manifests on disk, or writes bundle image metadata to disk. This subcommand has corresponding functionality to `opm alpha bundle build`, and is stable. Output and underlying behavior between these commands is the same, except nothing is written to disk unless `--generate-only` is set (unset by default). Refer to [`opm alpha bundle build` docs][registry-opm-build] for more information. CLI differences between these commands:
    * `opm alpha bundle build --default-channel=<channel-name>` equates to `operator-sdk bundle create --default=<channel-name>`.
    * `opm alpha bundle generate` equates to `operator-sdk bundle create --generate-only`.
    * `opm alpha bundle build --tag <image-tag>` equates to `operator-sdk bundle create <image-tag>`.
    * The `--overwrite` flag does not exist in `operator-sdk bundle create`.
* `bundle validate`: validates an Operator bundle image. This subcommand has corresponding functionality to `opm alpha bundle validate`, and is stable. Refer to the [`opm alpha bundle validate` docs][registry-opm-validate] for more information. One CLI difference between these commands:
    * `opm alpha bundle validate --tag <image-tag>` equates to `operator-sdk bundle validate <image-tag>`.


[sdk-generate-csv]:./generating-a-csv.md
[registry-bundle]:https://github.com/operator-framework/operator-registry/tree/v1.5.3#manifest-format
[registry-bundle-image]:https://github.com/operator-framework/operator-registry/blob/v1.5.3/docs/design/operator-bundle.md
[registry-opm-build]:https://github.com/operator-framework/operator-registry/blob/v1.5.9/docs/design/operator-bundle.md#build-bundle-image
[registry-opm-validate]:https://github.com/operator-framework/operator-registry/blob/v1.5.9/docs/design/operator-bundle.md#validate-bundle-image
