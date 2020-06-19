---
title: "operator-sdk bundle validate"
---
## operator-sdk bundle validate

Validate an operator bundle

### Synopsis

The 'operator-sdk bundle validate' command can validate both content and format of an operator bundle
image or an operator bundle directory on-disk containing operator metadata and manifests. This command will exit
with an exit code of 1 if any validation errors arise, and 0 if only warnings arise or all validators pass.

More information about operator bundles and metadata:
https://github.com/operator-framework/operator-registry/blob/master/docs/design/operator-bundle.md

NOTE: if validating an image, the image must exist in a remote registry, not just locally.


```
operator-sdk bundle validate [flags]
```

### Examples

```
The following command flow will generate test-operator bundle manifests and metadata,
then validate them for 'test-operator' version v0.1.0:

  # Generate manifests and metadata locally.
  $ make bundle

  # Validate the directory containing manifests and metadata.
  $ operator-sdk bundle validate ./bundle

To build and validate an image built with the above manifests and metadata:

  # Create a registry namespace or use an existing one.
  $ export NAMESPACE=<your registry namespace>

  # Build and push the image using the docker CLI.
  $ docker build -f bundle.Dockerfile -t quay.io/$NAMESPACE/test-operator:v0.1.0 .
  $ docker push quay.io/$NAMESPACE/test-operator:v0.1.0

  # Ensure the image with modified metadata and Dockerfile is valid.
  $ operator-sdk bundle validate quay.io/$NAMESPACE/test-operator:v0.1.0

```

### Options

```
  -h, --help                   help for validate
  -b, --image-builder string   Tool to pull and unpack bundle images. Only used when validating a bundle image. One of: [docker, podman, none] (default "docker")
```

### Options inherited from parent commands

```
      --verbose   Enable verbose logging
```

### SEE ALSO

* [operator-sdk bundle](../operator-sdk_bundle)	 - Manage operator bundle metadata

