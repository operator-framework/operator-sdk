## operator-sdk bundle validate

Validate an operator bundle image

### Synopsis

The 'operator-sdk bundle validate' command will validate both content and
format of an operator bundle image containing operator metadata and manifests.
This command will exit with a non-zero exit code if any validation tests fail.

Note: the image being validated must exist in a remote registry, not just locally.

```
operator-sdk bundle validate [flags]
```

### Examples

```
The following command flow will generate test-operator bundle image manifests
and validate that image:

$ cd ${HOME}/go/test-operator

# Generate manifests locally.
$ operator-sdk bundle build --generate-only

# Modify the metadata and Dockerfile.
$ cd ./deploy/olm-catalog/test-operator
$ vim ./metadata/annotations.yaml
$ vim ./Dockerfile

# Build and push the image using the docker CLI.
$ docker build -t quay.io/example/test-operator:v0.1.0 .
$ docker push quay.io/example/test-operator:v0.1.0

# Ensure the image with modified metadata/Dockerfile is valid.
$ operator-sdk bundle validate quay.io/example/test-operator:v0.1.0
```

### Options

```
  -h, --help                   help for validate
  -b, --image-builder string   Tool to extract container images. One of: [docker, podman] (default "docker")
```

### SEE ALSO

* [operator-sdk bundle](operator-sdk_bundle.md)	 - Work with operator bundle metadata and bundle images

