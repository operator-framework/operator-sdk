---
title: "operator-sdk bundle create"
---
## operator-sdk bundle create

Create an operator bundle image

### Synopsis

The 'operator-sdk bundle create' command will build an operator
bundle image containing operator metadata and manifests, tagged with the
provided image tag.

To write all files required to build a bundle image without building the
image, set '--generate-only=true'. A bundle.Dockerfile and bundle metadata
will be written if '--generate-only=true':

```
  $ operator-sdk bundle create --generate-only --directory ./deploy/olm-catalog/test-operator/manifests
  $ ls .
  ...
  bundle.Dockerfile
  ...
  $ tree ./deploy/olm-catalog/test-operator/
  ./deploy/olm-catalog/test-operator/
  ├── manifests
  │   ├── example.com_tests_crd.yaml
  │   └── test-operator.clusterserviceversion.yaml
  └── metadata
      └── annotations.yaml
```

'--generate-only' is useful if you want to build an operator's bundle image
manually or modify metadata before building an image.

More information about operator bundles and metadata:
https://github.com/operator-framework/operator-registry#manifest-format.

NOTE: bundle images are not runnable.


```
operator-sdk bundle create [flags]
```

### Examples

```
The following invocation will build a test-operator 0.1.0 bundle image using Docker.
This image will contain manifests for package channels 'stable' and 'beta':

  $ operator-sdk bundle create quay.io/example/test-operator:v0.1.0 \
      --directory ./deploy/olm-catalog/test-operator/manifests \
      --package test-operator \
      --channels stable,beta \
      --default-channel stable

Assuming your operator has the same name as your repo directory and the only
channel is 'stable', the above command can be abbreviated to:

  $ operator-sdk bundle create quay.io/example/test-operator:v0.1.0

The following invocation will generate test-operator bundle metadata and a
bundle.Dockerfile for your latest operator version without building the image:

  $ operator-sdk bundle create \
      --generate-only \
      --package test-operator \
      --channels beta \
      --default-channel beta

```

### Options

```
  -c, --channels string          The comma-separated list of channels that bundle image belongs to (default "stable")
  -e, --default-channel string   The default channel for the bundle image
  -d, --directory string         The directory where bundle manifests are located, ex. <project-root>/deploy/olm-catalog/test-operator/manifests
  -g, --generate-only            Generate metadata/, manifests/ and a Dockerfile on disk without building the bundle image
  -h, --help                     help for create
  -b, --image-builder string     Tool to build container images. One of: [docker, podman, buildah] (default "docker")
  -o, --output-dir string        Optional output directory for operator manifests
      --overwrite                Overwrite bundle.Dockerfile, manifests and metadata dirs if they exist. If --output-dir is also set, the original files will not be overwritten
  -p, --package string           The name of the package that bundle image belongs to. Set if package name differs from project name
  -t, --tag string               The path of a registry to pull from, image name and its tag that present the bundle image (e.g. quay.io/test/test-operator:v0.1.0)
```

### SEE ALSO

* [operator-sdk bundle](../operator-sdk_bundle)	 - Manage operator bundle metadata

