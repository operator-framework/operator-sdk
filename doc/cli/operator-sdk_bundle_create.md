## operator-sdk bundle create

Create an operator bundle image

### Synopsis

The 'operator-sdk bundle create' command will build an operator
bundle image containing operator metadata and manifests, tagged with the
provided image tag.

To write metadata and a bundle image Dockerfile to disk, set '--generate-only=true'.
Bundle metadata will be generated in <directory-arg>/metadata, and the Dockerfile
in <directory-arg>. This flag is useful if you want to build an operator's
bundle image manually or modify metadata before building an image.

More information on operator bundle images and metadata:
https://github.com/openshift/enhancements/blob/master/enhancements/olm/operator-bundle.md#docker

NOTE: bundle images are not runnable.

```
operator-sdk bundle create [flags]
```

### Examples

```
The following invocation will build a test-operator bundle image using Docker.
This image will contain manifests for package channels 'stable' and 'beta':

$ operator-sdk bundle create quay.io/example/test-operator:v0.1.0 \
    --directory ./deploy/olm-catalog/test-operator \
    --package test-operator \
    --channels stable,beta \
    --default-channel stable

Assuming your operator has the same name as your operator and the only channel
is 'stable', the above command can be abbreviated to:

$ operator-sdk bundle create quay.io/example/test-operator:v0.1.0

The following invocation will generate test-operator bundle metadata and
Dockerfile without building the image:

$ operator-sdk bundle create \
    --generate-only \
    --directory ./deploy/olm-catalog/test-operator \
    --package test-operator \
    --channels stable,beta \
    --default-channel stable
```

### Options

```
  -c, --channels strings         The list of channels that bundle image belongs to (default [stable])
  -e, --default-channel string   The default channel for the bundle image
  -d, --directory string         The directory where bundle manifests are located
  -g, --generate-only            Generate metadata and a Dockerfile on disk without building the bundle image
  -h, --help                     help for create
  -b, --image-builder string     Tool to build container images. One of: [docker, podman, buildah] (default "docker")
  -p, --package string           The name of the package that bundle image belongs to. Set if package name differs from project name (default "operator-sdk")
```

### SEE ALSO

* [operator-sdk bundle](operator-sdk_bundle.md)	 - Work with operator bundle metadata and bundle images

