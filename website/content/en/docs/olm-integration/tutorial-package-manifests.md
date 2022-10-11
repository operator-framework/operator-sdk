---
title: OLM Integration Package Manifests Quickstart
linkTitle: Package Manifests Quickstart
weight: 2
---
<!-- TODO(2.0.0): remove this document -->

**Note**
As operator framework has moved to using bundle format by default, the package manifest commands have been deprecated and will be removed soon. It is suggested that you follow the [bundle quickstart][quickstart-bundle] to package your operator. 

This guide assumes you have followed the introduction and *Setup* section of the [bundle quickstart][quickstart-bundle],
and have added the `packagemanifests` target to your `Makefile` as described [here][doc-packagemanifests-makefile].

**Important:** this guide assumes your project was scaffolded with `operator-sdk init --project-version=3`.
These features are unavailable to projects of version `2` or less; this information can be found by inspecting
your `PROJECT` file's `version` value.

## Creating package manifests

We will now create a package manifests format by running `make packagemanifests` in the root of the memcached-operator project:

```console
$ make packagemanifests
/home/user/go/bin/controller-gen rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
operator-sdk generate kustomize manifests -q
kustomize build config/manifests | operator-sdk generate packagemanifests -q --version 0.0.1
```

A versioned manifests directory `packagemanifests/0.0.1` containing a CSV and all CRDs in `config/crds` and a
package manifest YAML file `packagemanifests/<project-name>.package.yaml` have been created in the Operator project.

## Deploying an Operator with OLM

At this point in development we've generated all files necessary to build a memcached-operator registry.
Now we're ready to test the Operator with OLM.

### Testing package manifests

`operator-sdk run packagemanifests` will create an Operator [registry][operator-registry]
from manifests and metadata in the memcached-operator project, and inform OLM that memcached-operator v0.0.1
is ready to be deployed. This process effectively replicates production deployment in a constrained manner
to make sure OLM can deploy our Operator successfully before attempting real production deployment.

`run packagemanifests` performs some optionally configurable setup [under the hood][doc-testing-deployment], but for
most use cases the following invocation is all we need:

```console
$ operator-sdk run packagemanifests --version 0.0.1
INFO[0000] Running operator from directory packagemanifests
INFO[0000] Creating memcached-operator registry         
INFO[0000]   Creating ConfigMap "olm/memcached-operator-registry-manifests-package"
INFO[0000]   Creating ConfigMap "olm/memcached-operator-registry-manifests-0-0-1"
INFO[0000]   Creating Deployment "olm/memcached-operator-registry-server"
INFO[0000]   Creating Service "olm/memcached-operator-registry-server"
INFO[0000] Waiting for Deployment "olm/memcached-operator-registry-server" rollout to complete
INFO[0000]   Waiting for Deployment "olm/memcached-operator-registry-server" to rollout: 0 of 1 updated replicas are available
INFO[0066]   Deployment "olm/memcached-operator-registry-server" successfully rolled out
INFO[0066] Creating resources                           
INFO[0066]   Creating CatalogSource "default/memcached-operator-ocs"
INFO[0066]   Creating Subscription "default/memcached-operator-v0-0-1-sub"
INFO[0066]   Creating OperatorGroup "default/operator-sdk-og"
INFO[0066] Waiting for ClusterServiceVersion "default/memcached-operator.v0.0.1" to reach 'Succeeded' phase
INFO[0066]   Waiting for ClusterServiceVersion "default/memcached-operator.v0.0.1" to appear
INFO[0073]   Found ClusterServiceVersion "default/memcached-operator.v0.0.1" phase: Pending
INFO[0077]   Found ClusterServiceVersion "default/memcached-operator.v0.0.1" phase: InstallReady
INFO[0078]   Found ClusterServiceVersion "default/memcached-operator.v0.0.1" phase: Installing
INFO[0036]   Found ClusterServiceVersion "default/memcached-operator.v0.0.1" phase: Succeeded
INFO[0037] Successfully installed "memcached-operator.v0.0.1" on OLM version "0.15.1"

NAME                            NAMESPACE    KIND                        STATUS
memcacheds.cache.example.com    default      CustomResourceDefinition    Installed
memcached-operator.v0.0.1       default      ClusterServiceVersion       Installed
```

As long as both the `ClusterServiceVersion` and all `CustomResourceDefinition`'s return an `Installed` status,
the memcached-operator has been deployed successfully.

Now that we're done testing the memcached-operator, we should probably clean up the Operator's resources.
[`operator-sdk cleanup`][cli-cleanup] will do this for you:

```console
$ operator-sdk cleanup memcached-operator
INFO[0000] subscription "memcached-operator-v0-0-1-sub" deleted
INFO[0000] customresourcedefinition "memcacheds.cache.example.com" deleted
INFO[0000] clusterserviceversion "memcached-operator.v0.0.1" deleted
INFO[0000] clusterrole "memcached-operator-metrics-reader" deleted
INFO[0000] serviceaccount "default" deleted
INFO[0000] role "memcached-operator.v0.0.1-jhjk7" deleted
INFO[0000] rolebinding "memcached-operator.v0.0.1-jhjk7-default-mxv6m" deleted
INFO[0000] catalogsource "memcached-operator-ocs" deleted
INFO[0000] operatorgroup "operator-sdk-og" deleted
INFO[0001] operator "memcached-operator" uninstalled
```

## Migrating packagemanifests to bundles

In order to migrate packagemanifests to bundles, `operator-sdk pkgman-to-bundle` command can be used.

As an example, consider the packagemanifests directory to have the following structure:

```
packagemanifests
└── etcd
    ├── 0.0.1
    │   ├── etcdcluster.crd.yaml
    │   └── etcdoperator.clusterserviceversion.yaml
    ├── 0.0.2
    │   ├── etcdbackup.crd.yaml
    │   ├── etcdcluster.crd.yaml
    │   ├── etcdoperator.v0.0.2.clusterserviceversion.yaml
    │   └── etcdrestore.crd.yaml
    └── etcd.package.yaml
```

Here, we have manifests for two versions of the `etcd` operator. The following command will generate bundles for each of these versions.

```console
$ operator-sdk pkgman-to-bundle packagemanifests --output-dir etcd-bundle/
INFO[0000] Packagemanifests will be migrated to bundles in bundle directory
INFO[0000] Creating etcd-bundle/bundle-0.0.1/bundle.Dockerfile
INFO[0000] Creating etcd-bundle/bundle-0.0.1/metadata/annotations.yaml
...
```

This will create output bundles in the directory `etcd-bundle`. The output directory will look like:

```
etcd-bundle/
├── bundle-0.0.1
│   ├── bundle
│   │   ├── manifests
│   │   │   ├── etcdcluster.crd.yaml
│   │   │   ├── etcdoperator.clusterserviceversion.yaml
│   │   ├── metadata
│   │   │   └── annotations.yaml
│   │   └── tests
│   │       └── scorecard
│   │           └── config.yaml
│   └── bundle.Dockerfile
└── bundle-0.0.2
    ├── bundle
    │   ├── manifests
    │   │   ├── etcdbackup.crd.yaml
    │   │   ├── etcdcluster.crd.yaml
    │   │   ├── etcdoperator.v0.0.2.clusterserviceversion.yaml
    │   │   ├── etcdrestore.crd.yaml
    │   └── metadata
    │       └── annotations.yaml
    └── bundle.Dockerfile
```

To build images for the bundles, the base container image name can be provided using `--image-tag-base` flag. This name should be provided without the tag (`:` and characters following), as the command will tag each bundle image with its packagemanifests directory name, i.e. `<image-tag-base>:<dir-name>`. For example, the following command for the above `packagemnifests` directory would build the bundles `quay.io/example/etcd-bundle:0.0.1` and `quay.io/example/etcd-bundle:0.0.2`.

```sh
operator-sdk pkgman-to-bundle packagemanifests --image-tag-base quay.io/example/etcd-bundle
```

A custom command can also be specified to build images, using the `--build-cmd` flag. The default command is `docker build`. For example:

```console
$ operator-sdk pkgman-to-bundle packagemanifests --output-dir etcd-bundle/ --image-tag-base quay.io/example/etcd --build-cmd "podman build -f bundle.Dockerfile . -t"
```

However, if using a custom command, it needs to be made sure that the command is in the `PATH` or a fully qualified path name is provided as input to the flag.

Once the command has finished building your bundle images and they have been added to a catalog image, delete all bundle directories except for the latest one. This directory will contain manifests for your operator's head bundle, and should be versioned with version control system like git. Move this directory and its `bundle.Dockerfile` to your project's root:

```console
$ cp -r ./etcd-bundle/bundle-0.0.2/* .
$ rm -rf ./etcd-bundle
```

Try building then running your bundle on a live cluster to make sure it works as expected:

```console
$ make bundle bundle-build bundle-push
$ operator-sdk run bundle quay.io/example/etcd-bundle:0.0.2
```

[quickstart-bundle]:/docs/olm-integration/quickstart-bundle
[operator-registry]:https://github.com/operator-framework/operator-registry
[cli-cleanup]:/docs/cli/operator-sdk_cleanup
[doc-packagemanifests-makefile]:/docs/olm-integration/generation/#package-manifests-format
[doc-testing-deployment]:/docs/olm-integration/testing-deployment
