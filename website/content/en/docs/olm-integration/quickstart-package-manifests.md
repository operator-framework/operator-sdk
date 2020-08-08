---
title: OLM Integration Package Manifests Quickstart
linkTitle: Package Manifests Quickstart
weight: 2
---

This guide assumes you have followed the introduction and *Setup* section of the [bundle quickstart][quickstart-bundle],
and have added the `packagemanifests` target to your `Makefile` as described [here][doc-olm-generate].

**Important:** this guide assumes your project was scaffolded with `operator-sdk init --project-version=3-alpha`.
These features are unavailable to projects of version `2` or less; this information can be found by inspecting
your `PROJECT` file's `version` value.

## Creating package manifests

We will now create a package manifests format by running `make packagemanifests` in the root of the memcached-operator project:

```console
$ make packagemanifests
/home/user/go/bin/controller-gen "crd:trivialVersions=true" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
operator-sdk generate kustomize manifests -q
kustomize build config/manifests | operator-sdk generate packagemanifests -q --version 0.0.1
```

A versioned manifests directory `packagemanifests/0.0.1` containing a CSV and all CRDs in `config/crds` and a
package manifest YAML file `packagemanifests/<project-name>.package.yaml` have been created in the Operator project.

## Deploying an Operator with OLM

At this point in development we've generated all files necessary to build a memcached-operator registry.
Now we're ready to test the Operator with OLM.

### Testing package manifests

[`operator-sdk run packagemanifests`][cli-run-packagemanifests] will create an Operator [registry][operator-registry]
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


[quickstart-bundle]:/docs/olm-integration/quickstart-bundle
[operator-registry]:https://github.com/operator-framework/operator-registry
[cli-run-packagemanifests]:/docs/cli/operator-sdk_run_packagemanifests
[cli-cleanup]:/docs/cli/operator-sdk_cleanup
[doc-olm-generate]:/docs/olm-integration/generation#overview
[doc-testing-deployment]:/docs/olm-integration/testing-deployment
