---
title: Helm Operator Tutorial
linkTitle: Tutorial
weight: 200
description: An in-depth walkthrough of building and running a Helm-based operator.
---

**NOTE:** If your project was created with an `operator-sdk` version prior to `v1.0.0`
please [migrate][migration-guide], or consult the [legacy docs][legacy-quickstart-doc].

## Prerequisites

- Go through the [installation guide][install-guide].
- Make sure your user is authorized with `cluster-admin` permissions.
- An accessible image registry for various operator images (ex. [hub.docker.com](https://hub.docker.com/signup),
[quay.io](https://quay.io/)) and be logged in to your command line environment.
  - `example.com` is used as the registry Docker Hub namespace in these examples.
  Replace it with another value if using a different registry or namespace.
  - [Authentication and certificates][image-reg-config] if the registry is private or uses a custom CA.

## Overview

We will create a sample project to let you know how it works and this sample will:

- Create an Nginx Deployment if it doesn't exist
- Ensure that the Deployment size is the same as specified by the Nginx CR spec

## Create a new project

Use the CLI to create a new Helm-based nginx-operator project:

```sh
mkdir nginx-operator
cd nginx-operator
operator-sdk init --plugins helm --domain example.com --group demo --version v1alpha1 --kind Nginx
```

This creates the nginx-operator project specifically for watching the
Nginx resource with APIVersion `demo.example.com/v1alpha1` and Kind
`Nginx`.

For Helm-based projects, `operator-sdk init` also generates the RBAC rules
in `config/rbac/role.yaml` based on the resources that would be deployed by the
chart's default manifest. Be sure to double check that the rules generated
in `config/rbac/role.yaml` meet the operator's permission requirements.

To learn more about the project directory structure, see the
[project layout][layout-doc] doc.

### Use an existing chart

Instead of creating your project with a boilerplate Helm chart, you can also use `--helm-chart`, `--helm-chart-repo`, and `--helm-chart-version` to use an existing chart, either from your local filesystem or a remote chart repository.

If `--helm-chart` is specified, the `--group`, `--version`, and `--kind` flags become optional. If left unset, the default will be:

| Flag | Value |
| :--- | :---    |
| domain | my.domain |
| group | charts |
| kind |  deduce from the specified chart |
| version | v1alpha1 |

If `--helm-chart` is a local chart archive (e.g `example-chart-1.2.0.tgz`) or directory,
it will be validated and unpacked or copied into the project.

Otherwise, the SDK will attempt to fetch the specified helm chart from a remote repository.

If a custom repository URL is not specified by `--helm-chart-repo`, the following chart reference formats are supported:

- `<repoName>/<chartName>`: Fetch the helm chart named `chartName` from the helm
                            chart repository named `repoName`, as specified in the
                           `$HELM_HOME/repositories/repositories.yaml` file.
                            Use [`helm repo add`](https://helm.sh/docs/helm/helm_repo_add) to configure this file.

- `<url>`: Fetch the helm chart archive at the specified URL.

If a custom repository URL is specified by `--helm-chart-repo`, the only supported format for `--helm-chart` is:

- `<chartName>`: Fetch the helm chart named `chartName` in the helm chart repository
                 specified by the `--helm-chart-repo` URL.

If `--helm-chart-version` is not set, the SDK will fetch the latest available version of the helm chart. Otherwise, it will fetch the specified version. The option `--helm-chart-version` is not used when `--helm-chart` itself refers to a specific version, for example when it is a local path or a URL.

**Note:** For more details and examples run `operator-sdk init --plugins helm --help`.

<!--
todo(camilamacedo86): Create an Ansible operator scope document.
https://github.com/operator-framework/operator-sdk/issues/3447
-->


## Customize the operator logic

For this example the nginx-operator will execute the following
reconciliation logic for each `Nginx` Custom Resource (CR):

- Create an nginx Deployment if it doesn't exist
- Create an nginx Service if it doesn't exist
- Create an nginx Ingress if it is enabled and doesn't exist
- Ensure that the Deployment, Service, and optional Ingress match the desired configuration (e.g. replica count, image, service type, etc) as specified by the `Nginx` CR

### Watch the Nginx CR

By default, the nginx-operator watches `Nginx` resource events as shown
in `watches.yaml` and executes Helm releases using the specified chart:

```yaml
# Use the 'create api' subcommand to add watches to this file.
- group: demo
  version: v1alpha1
  kind: Nginx
  chart: helm-charts/nginx
#+kubebuilder:scaffold:watch
```

### Reviewing the Nginx Helm Chart

When a Helm operator project is created, the SDK creates an example Helm chart
that contains a set of templates for a simple Nginx release.

For this example, we have templates for deployment, service, and ingress
resources, along with a `NOTES.txt` template, which Helm chart developers use
to convey helpful information about a release.

If you aren't already familiar with Helm Charts, take a moment to review
the [Helm Chart developer documentation][helm-charts].

### Understanding the Nginx CR spec

Helm uses a concept called [values][helm-values] to provide customizations
to a Helm chart's defaults, which are defined in the Helm chart's `values.yaml`
file.

Overriding these defaults is as simple as setting the desired values in the CR
spec. Let's use the number of replicas as an example.

First, inspecting `helm-charts/nginx/values.yaml`, we see that the chart has a
value called `replicaCount` and it is set to `1` by default. If we want to have
2 nginx instances in our deployment, we would need to make sure our CR spec
contained `replicaCount: 2`.

Update `config/samples/demo_v1alpha1_nginx.yaml` to look like the following:

```yaml
apiVersion: demo.example.com/v1alpha1
kind: Nginx
metadata:
  name: nginx-sample
spec:
  replicaCount: 2
```

Similarly, we see that the default service port is set to `80`, but we would
like to use `8080`, so we'll again update `config/samples/demo_v1alpha1_nginx.yaml`
by adding the service port override:

```yaml
apiVersion: demo.example.com/v1alpha1
kind: Nginx
metadata:
  name: nginx-sample
spec:
  replicaCount: 2
  service:
    port: 8080
```

As you may have noticed, the Helm operator simply applies the entire spec as if
it was the contents of a values file, just like `helm install -f ./overrides.yaml`
works.

## Configure the operator's image registry

All that remains is to build and push the operator image to the desired image registry.
Your Makefile composes image tags either from values written at project initialization or from the CLI.
In particular, `IMAGE_TAG_BASE` lets you define a common image registry, namespace, and partial name
for all your image tags. Update this to another registry and/or namespace if the current value is incorrect.
Afterwards you can update the `IMG` variable definition like so:

```diff
-IMG ?= controller:latest
+IMG ?= $(IMAGE_TAG_BASE):$(VERSION)
```

Once done, you do not have to set `IMG` or any other image variable in the CLI. The following command will
build and push an operator image tagged as `example.com/nginx-operator:v0.0.1` to Docker Hub:

```console
make docker-build docker-push
```


## Run the operator

There are three ways to run the operator:

- As a Go program outside a cluster
- As a Deployment inside a Kubernetes cluster
- Managed by the [Operator Lifecycle Manager (OLM)][doc-olm] in [bundle][tutorial-bundle] format

### 1. Run locally outside the cluster

Execute the following command, which installs your CRDs and runs the manager locally:

```sh
make install run
```

### 2. Run as a Deployment inside the cluster

By default, a new namespace is created with the name `<project-name>-system`, ex. `nginx-operator-system`, and will be used for the deployment.

Run the following to deploy the operator. This will also install the RBAC manifests from `config/rbac`.

```sh
make deploy
```

Verify that the nginx-operator is up and running:

```console
$ kubectl get deployment -n nginx-operator-system
NAME                                    READY   UP-TO-DATE   AVAILABLE   AGE
nginx-operator-controller-manager   1/1     1            1           8m
```

### 3. Deploy your Operator with OLM

First, install [OLM][doc-olm]:

```sh
operator-sdk olm install
```

Bundle your operator, then build and push the bundle image. The `bundle` target generates a [bundle][doc-bundle]
in the `bundle` directory containing manifests and metadata defining your operator.
`bundle-build` and `bundle-push` build and push a bundle image defined by `bundle.Dockerfile`.

```sh
make bundle bundle-build bundle-push
```

Finally, run your bundle. If your bundle image is hosted in a registry that is private and/or
has a custom CA, these [configuration steps][image-reg-config] must be completed.

```sh
operator-sdk run bundle example.com/nginx-operator-bundle:v0.0.1
```

Check out the [docs][tutorial-bundle] for a deep dive into the `operator-sdk`'s OLM integration.


## Create a Nginx CR

Create the nginx CR that we modified earlier:

```sh
kubectl apply -f config/samples/demo_v1alpha1_nginx.yaml
```

Ensure that the nginx-operator creates the deployment for the CR:

```console
$ kubectl get deployment
NAME           READY   UP-TO-DATE   AVAILABLE   AGE
nginx-sample   2/2     2            2           2m13s
```

Check the pods to confirm 2 replicas were created:

```console
$ kubectl get pods
NAME                                                   READY   STATUS    RESTARTS   AGE
nginx-sample-c786bfdcf-4g6md                           1/1     Running   0          81s
nginx-sample-c786bfdcf-6bhmx                           1/1     Running   0          81s
```

Check that the service port is set to `8080`:

```console
$ kubectl get service
NAME                                      TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)    AGE
nginx-sample                              ClusterIP   10.96.26.3   <none>        8080/TCP   1m
```

### Update the replicaCount and remove the port

Change the `spec.replicaCount` field from 2 to 3, remove the `spec.service`
field:

```console
$ cat config/samples/demo_v1alpha1_nginx.yaml
apiVersion: demo.example.com/v1alpha1
kind: Nginx
metadata:
  name: nginx-sample
spec:
  replicaCount: 3
```

And apply the change:

```sh
kubectl apply -f config/samples/demo_v1alpha1_nginx.yaml
```

Confirm that the operator changes the deployment size:

```console
$ kubectl get deployment
NAME                                           DESIRED   CURRENT   UP-TO-DATE     AGE
nginx-sample                                   3/3       3            3           7m29s
```

Check that the service port is set to the default (`80`):

```console
$ kubectl get service
NAME                                      TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)  AGE
nginx-sample                              ClusterIP   10.96.152.76    <none>        80/TCP   7m54s
```

### Troubleshooting

Use the following command to check the operator logs.

```sh
kubectl logs deployment.apps/nginx-operator-controller-manager  -n nginx-operator-system -c manager
```

Use the following command to check the CR status and events.

```sh
kubectl describe nginxes.demo.example.com
```

### Cleanup

Clean up the resources:

```sh
kubectl delete -f config/samples/demo_v1alpha1_nginx.yaml
```

**Note:** Make sure the above custom resource has been deleted before proceeding to
run `make undeploy`, as helm-operator's controller adds finalizers to the custom resources.
Otherwise your cluster may have dangling custom resource objects that cannot be deleted.

```sh
make undeploy
```

## Next steps

Next, check out the following:
1. Operator packaging and distribution with [OLM][olm-integration].
1. The [advanced features][advanced-features] doc for more use cases and under-the-hood details.


[legacy-quickstart-doc]:https://github.com/operator-framework/operator-sdk/tree/v0.19.x/website/content/en/docs/helm/quickstart.md
[migration-guide]:/docs/building-operators/helm/migration
[install-guide]:/docs/building-operators/helm/installation
[image-reg-config]:/docs/olm-integration/cli-overview#private-bundle-and-catalog-image-registries
[layout-doc]: /docs/overview/project-layout
[helm-charts]:https://helm.sh/docs/topics/charts/
[helm-values]:https://helm.sh/docs/intro/using_helm/#customizing-the-chart-before-installing
[helm-official]:https://helm.sh/docs/
[tutorial-bundle]:/docs/olm-integration/tutorial-bundle
[doc-olm]:/docs/olm-integration/tutorial-bundle/#enabling-olm
[olm-integration]:/docs/olm-integration
[advanced-features]:/docs/building-operators/helm/reference/advanced_features
