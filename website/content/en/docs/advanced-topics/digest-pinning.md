---
title: Digest Pinning
weight: 80
---

# Image Digest Pinning

Operator authors have the ability to pin container images by their
[digest](https://github.com/opencontainers/image-spec/blob/main/descriptor.md) when generating
bundles. The digest simultaneously acts as a unique identifier for the image as well as a checksum
for the image contents. Referencing images by digest rather than their tag ensures the operator
bundle deployment is consistent and reproducible.

## Usage

To generate bundles that reference images by digest, pass the `--use-image-digests` flag to operator-sdk:

```sh
$ operator-sdk generate bundle --use-image-digests
```

Operator projects using the `go` and `helm` builders can also set the `USE_IMAGE_DIGESTS` Makefile variable to `true`:

```sh
$ make bundle USE_IMAGE_DIGESTS=true
```

## Bundle Image Detection and Resolution

`operator-sdk` resolves image references to digests by analyzing the `ClusterServiceVersion` object
provided as input. The following fields in the CSV are used to find and resolve image references:

- All containers in the CSV deployments (`spec.install.spec.deployments`).
- All environment variables prefixed with `RELATED_IMAGE_` and have a valid container image reference.

Each resolved image is rendered into the ouput bundle's `ClusterServiceVersion` as follows:

1. Images referenced by tag are updated to be referenced by image digest SHA.
2. Each resolved image is also referenced in the `spec.relatedImages` field in the bundle CSV.

The `relatedImages` field is intended for external tools to identify all container images needed to
deploy your operator and operands. It is not required to bundle or deploy your operator.
