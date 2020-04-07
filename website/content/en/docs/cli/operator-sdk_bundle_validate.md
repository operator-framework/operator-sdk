---
title: "operator-sdk bundle validate"
---
## operator-sdk bundle validate

Validate an operator bundle image

### Synopsis

The 'operator-sdk bundle validate' command can validate both content and
format of an operator bundle image or an operator bundles directory on-disk
containing operator metadata and manifests. This command will exit with a non-zero
exit code if any validation tests fail.

Note: if validating an image, the image must exist in a remote registry, not
just locally.


```
operator-sdk bundle validate [flags]
```

### Examples

```
The following command flow will generate test-operator bundle image manifests
and validate them, assuming a bundle for 'test-operator' version v0.1.0 exists at
<project-root>/deploy/olm-catalog/test-operator/0.1.0:

  # Generate manifests locally.
  $ operator-sdk bundle create \
      --generate-only \
      --directory ./deploy/olm-catalog/test-operator/0.1.0

  # Validate the directory containing manifests and metadata.
  $ operator-sdk bundle validate ./deploy/olm-catalog/test-operator

To build and validate an image:

  # Build and push the image using the docker CLI.
	$ operator-sdk bundle create quay.io/example/test-operator:v0.1.0 \
      --directory ./deploy/olm-catalog/test-operator/0.1.0
  $ docker push quay.io/example/test-operator:v0.1.0

  # Ensure the image with modified metadata and Dockerfile is valid.
  $ operator-sdk bundle validate quay.io/example/test-operator:v0.1.0


```

### Options

```
  -h, --help                   help for validate
  -b, --image-builder string   Tool to extract container images. One of: [docker, podman] (default "docker")
```

### SEE ALSO

* [operator-sdk bundle](../operator-sdk_bundle)	 - Work with operator bundle metadata and bundle images

