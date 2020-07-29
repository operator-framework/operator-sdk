---
title: Tutorial for Helm-based Operators
linkTitle: Tutorial
weight: 200
description: An in-depth walkthough that demonstrates how to build and run a Helm-based operator.
---

## Overview

This guide walks through an example of building a simple nginx-operator powered by [Helm][helm-official] using tools and libraries provided by the Operator SDK.

## Prerequisites

- [Install `operator-sdk`][operator_install] and its prequisites.
- Access to a Kubernetes v1.16.0+ cluster.

## Create a new project

Use the CLI to create a new Helm-based nginx-operator project:

```sh
mkdir nginx-operator
cd nginx-operator
operator-sdk init --plugins=helm --domain=com --group=example --version=v1alpha1 --kind=Nginx
```

This creates the nginx-operator project specifically for watching the
Nginx resource with APIVersion `example.com/v1alpha1` and Kind
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

**Note:** For more details and examples run `operator-sdk init --plugins=helm --help`.

### Operator scope

Read the [operator scope][operator-scope] documentation on how to run your operator as namespace-scoped vs cluster-scoped.


## Customize the operator logic

For this example the nginx-operator will execute the following
reconciliation logic for each `Nginx` Custom Resource (CR):

- Create a nginx Deployment if it doesn't exist
- Create a nginx Service if it doesn't exist
- Create a nginx Ingress if it is enabled and doesn't exist
- Ensure that the Deployment, Service, and optional Ingress match the desired configuration (e.g. replica count, image, service type, etc) as specified by the `Nginx` CR

### Watch the Nginx CR

By default, the nginx-operator watches `Nginx` resource events as shown
in `watches.yaml` and executes Helm releases using the specified chart:

```yaml
# Use the 'create api' subcommand to add watches to this file.
- group: example.com
  version: v1alpha1
  kind: Nginx
  chart: helm-charts/nginx
# +kubebuilder:scaffold:watch
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

Update `config/samples/example_v1alpha1_nginx.yaml` to look like the following:

```yaml
apiVersion: example.com/v1alpha1
kind: Nginx
metadata:
  name: nginx-sample
spec:
  replicaCount: 2
```

Similarly, we see that the default service port is set to `80`, but we would
like to use `8080`, so we'll again update `config/samples/example_v1alpha1_nginx.yaml`
by adding the service port override:

```yaml
apiVersion: example.com/v1alpha1
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

## Build and run the operator

Before running the operator, Kubernetes needs to know about the new custom
resource definition the operator will be watching.

Deploy the CRD:

```sh
make install
```

Once this is done, there are two ways to run the operator:

- As a pod inside a Kubernetes cluster
- As a go program outside the cluster using `operator-sdk`

### 1. Run as a pod inside a Kubernetes cluster

Running as a pod inside a Kubernetes cluster is preferred for production use.

Build the nginx-operator image and push it to a registry:

```sh
export IMG=quay.io/example/nginx-operator:v0.0.1
make docker-build docker-push IMG=$IMG
```

**Note:** Kubernetes deployment manifests are generated in `config/manager/manager.yaml`.

Deploy the nginx-operator:

```sh
make deploy IMG=$IMG
```

Verify that the nginx-operator is up and running:

```sh
$ kubectl get deployment -n nginx-operator-system
NAME                                DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
nginx-operator-controller-manager   1/1       1         1            1           77s
```

### 2. Run outside the cluster

This method is preferred during the development cycle to speed up deployment and testing.

Run the operator locally with the default Kubernetes config file present at
`$HOME/.kube/config`:

```sh
$ make run
INFO[0000] Go Version: go1.10.3
INFO[0000] Go OS/Arch: linux/amd64
INFO[0000] operator-sdk Version: v0.1.1+git
```

Run the operator locally with a provided Kubernetes config file:

```sh
$ operator-sdk run local --kubeconfig=<path_to_config>
INFO[0000] Go Version: go1.10.3
INFO[0000] Go OS/Arch: linux/amd64
INFO[0000] operator-sdk Version: v0.2.0+git
```

### 3. Deploy your Operator with the Operator Lifecycle Manager (OLM)

OLM will manage creation of most if not all resources required to run your operator,
using a bit of setup from other `operator-sdk` commands. Check out the OLM integration
[user guide][quickstart-bundle] for more information.

## Deploy the Nginx custom resource

Apply the nginx CR that we modified earlier:

```sh
kubectl apply -f config/samples/example_v1alpha1_nginx.yaml
```

Ensure that the nginx-operator creates the deployment for the CR:

```sh
$ kubectl get deployment
NAME           READY   UP-TO-DATE   AVAILABLE   AGE
nginx-sample   2/2     2            2           2m13s
```

Check the pods to confirm 2 replicas were created:

```sh
$ kubectl get pods
NAME                                                   READY   STATUS    RESTARTS   AGE
nginx-sample-c786bfdcf-4g6md                           1/1     Running   0          81s
nginx-sample-c786bfdcf-6bhmx                           1/1     Running   0          81s

Check that the service port is set to `8080`:

```sh
$ kubectl get service
NAME                                      TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)    AGE
nginx-sample                              ClusterIP   10.96.26.3   <none>        8080/TCP   1m
```

### Update the replicaCount and remove the port

Change the `spec.replicaCount` field from 2 to 3, remove the `spec.service`
field:

```sh
$ cat config/samples/example_v1alpha1_nginx.yaml
apiVersion: example.com/v1alpha1
kind: Nginx
metadata:
  name: nginx-sample
spec:
  replicaCount: 3
```

And apply the change:

```sh
kubectl apply -f config/samples/example_v1alpha1_nginx.yaml
```

Confirm that the operator changes the deployment size:

```sh
$ kubectl get deployment
NAME                                           DESIRED   CURRENT   UP-TO-DATE     AGE
nginx-sample                                   3/3       3            3           7m29s
```

Check that the service port is set to the default (`80`):

```sh
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
kubectl describe nginxes.example.com 
```

### Cleanup

Clean up the resources:

```sh
kubectl delete -f config/samples/example_v1alpha1_nginx.yaml
make undeploy
```
**NOTE** Additional CR/CRD's can be added to the project by running, for example, the command :`operator-sdk create api --group=example --version=v1alpha1 --kind=AppService`

<!--
todo(camilamacedo86): https://github.com/operator-framework/operator-sdk/issues/3447 
-->
[operator-scope]: /docs/building-operators/golang/operator-scope
[layout-doc]: /docs/building-operators/helm/reference/project_layout/
[helm-charts]:https://helm.sh/docs/topics/charts/
[helm-values]:https://helm.sh/docs/intro/using_helm/#customizing-the-chart-before-installing
[helm-official]:https://helm.sh/docs/
[operator_install]: /docs/installation/install-operator-sdk
