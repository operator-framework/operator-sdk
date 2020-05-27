---
title: "operator-sdk bundle validate"
---
## operator-sdk bundle validate

Validate an operator bundle image

### Synopsis

The 'operator-sdk bundle validate' command can validate both content and
format of an operator bundle image or an operator bundles directory on-disk
containing operator metadata and manifests. This command will exit with an
exit code of 1 if any validation errors arise, and 0 if only warnings arise or
all validators pass.

More information about operator bundles and metadata:
https://github.com/operator-framework/operator-registry#manifest-format.

NOTE: if validating an image, the image must exist in a remote registry, not
just locally.


```
operator-sdk bundle validate [flags]
```

### Examples

```
The following command flow will generate test-operator bundle image manifests
and validate them, assuming a bundle for 'test-operator' version v0.1.0 exists at
<project-root>/deploy/olm-catalog/test-operator/manifests:

  # Generate manifests locally.
  $ operator-sdk bundle create --generate-only

  # Validate the directory containing manifests and metadata.
  $ operator-sdk bundle validate ./deploy/olm-catalog/test-operator

To build and validate an image:

  # Create a registry namespace or use an existing one.
  $ export NAMESPACE=<your registry namespace>

  # Build and push the image using the docker CLI.
  $ operator-sdk bundle create quay.io/$NAMESPACE/test-operator:v0.1.0
  $ docker push quay.io/$NAMESPACE/test-operator:v0.1.0

  # Ensure the image with modified metadata and Dockerfile is valid.
  $ operator-sdk bundle validate quay.io/$NAMESPACE/test-operator:v0.1.0


```

### Options

```
  -h, --help                   help for validate
  -b, --image-builder string   Tool to extract bundle image data. Only used when validating a bundle image. One of: [docker, podman] (default "docker")
```

### SEE ALSO

* [operator-sdk bundle](../operator-sdk_bundle)	 - Manage operator bundle metadata

