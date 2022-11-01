---
title: OLM Integration Bundle Quickstart
linkTitle: Bundle Quickstart
weight: 1
---

The [Operator Lifecycle Manager (OLM)][olm] is a set of cluster resources that manage the lifecycle of an Operator.
The Operator SDK supports both creating manifests for OLM deployment, and testing your Operator on an OLM-enabled
Kubernetes cluster.

This document is intended to quickly walk through the steps to generate an OLM bundle. For further explanation,
or if you're using package manifests, see the [Bundle Tutorial][tutorial-bundle].

**Important:** this guide assumes your project was scaffolded with `operator-sdk init --project-version=3`.
These features are unavailable to projects of version `2` or less; this information can be found by inspecting
your `PROJECT` file's `version` value.

## Prerequisites

- Have a working operator that you have uploaded to a container registry. This guide assumes the simple Golang Memcached operator from [the building operators section][sdk-user-guide-go] at version `0.0.1`.
- Make sure your user is authorized with `cluster-admin` permissions.
- Have OLM installed on your cluster. The command `operator-sdk olm install` will attempt to install a basic OLM deployment on your cluster.

## Steps

1. Export environment variables

```sh
$ export USERNAME=<container-registry-username>
$ export VERSION=0.0.1
$ export IMG=docker.io/$USERNAME/memcached-operator:v$VERSION // location where your operator image is hosted
$ export BUNDLE_IMG=docker.io/$USERNAME/memcached-operator-bundle:v$VERSION // location where your bundle will be hosted
```

- Create a bundle from the root directory of your project

```sh
$ make bundle
```

This will prompt you to enter basic information about your operator.

- Build and push the bundle image

```sh
$ make bundle-build bundle-push
```

- Validate the bundle

```sh
$ operator-sdk bundle validate $BUNDLE_IMG
```

- Install the bundle with OLM

```sh
$ operator-sdk run bundle $BUNDLE_IMG
```

## Next Steps

Read the [full tutorial][tutorial-bundle] for a more in-depth look at creating and using a bundle.

[tutorial-bundle]:/docs/olm-integration/tutorial-bundle
[sdk-user-guide-go]:/docs/building-operators/golang/quickstart
[olm]:https://github.com/operator-framework/operator-lifecycle-manager/
