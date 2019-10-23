# Helm User Guide for Operator SDK

This guide walks through an example of building a simple nginx-operator powered by Helm using tools and libraries provided by the Operator SDK.

## Prerequisites

- [git][git-tool]
- [docker][docker-tool] version 17.03+.
- [kubectl][kubectl-tool] version v1.11.3+.
- [go][go-tool] version v1.13+. (Optional if you aren't installing from source)
- Access to a Kubernetes v1.11.3+ cluster.

**Note**: This guide uses [minikube][minikube-tool] version v0.25.0+ as the
local Kubernetes cluster and [quay.io][quay-link] for the public registry.

## Install the Operator SDK CLI

Follow the steps in the [installation guide][install-guide] to learn how to install the Operator SDK CLI tool.

## Create a new project

Use the CLI to create a new Helm-based nginx-operator project:

```sh
operator-sdk new nginx-operator --api-version=example.com/v1alpha1 --kind=Nginx --type=helm
cd nginx-operator
```

This creates the nginx-operator project specifically for watching the
Nginx resource with APIVersion `helm.example.com/v1alpha1` and Kind
`Nginx`.

For Helm-based projects, `operator-sdk new` also generates the RBAC rules
in `deploy/role.yaml` based on the resources that would be deployed by the
chart's default manifest. Be sure to double check that the rules generated
in `deploy/role.yaml` meet the operator's permission requirements.

To learn more about the project directory structure, see the
[project layout][layout-doc] doc.

### Use an existing chart

Instead of creating your project with a boilerplate Helm chart, you can also use `--helm-chart`, `--helm-chart-repo`, and `--helm-chart-version` to use an existing chart, either from your local filesystem or a remote chart repository.

If `--helm-chart` is specified, `--api-version` and `--kind` become optional. If left unset, the SDK will default `--api-version` to `charts.helm.k8s.io/v1alpha1` and will deduce `--kind` from the specified chart.

If `--helm-chart` is a local chart archive or directory, it will be validated and unpacked or copied into the project.

Otherwise, the SDK will attempt to fetch the specified helm chart from a remote repository.

If a custom repository URL is not specified by `--helm-chart-repo`, the following chart reference formats are supported:

- `<repoName>/<chartName>`: Fetch the helm chart named `chartName` from the helm
                            chart repository named `repoName`, as specified in the
                            $HELM_HOME/repositories/repositories.yaml file.

- `<url>`: Fetch the helm chart archive at the specified URL.

If a custom repository URL is specified by `--helm-chart-repo`, the only supported format for `--helm-chart` is:

- `<chartName>`: Fetch the helm chart named `chartName` in the helm chart repository
                 specified by the `--helm-chart-repo` URL.

If `--helm-chart-version` is not set, the SDK will fetch the latest available version of the helm chart. Otherwise, it will fetch the specified version. `--helm-chart-version` is not used when `--helm-chart` itself refers to a specific version, for example when it is a local path or a URL.

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
---
- version: v1alpha1
  group: helm.example.com
  kind: Nginx
  chart: /opt/helm/helm-charts/nginx
```

### Reviewing the Nginx Helm Chart

When a Helm operator project is created, the SDK creates an example Helm chart
that contains a set of templates for a simple Nginx release.

For this example, we have templates for deployment, service, and ingress
resources, along with a NOTES.txt template, which Helm chart developers use
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

Update `deploy/crds/helm.example.com_v1alpha1_nginx_cr.yaml` to look like the following:

```yaml
apiVersion: helm.example.com/v1alpha1
kind: Nginx
metadata:
  name: example-nginx
spec:
  replicaCount: 2
```

Similarly, we see that the default service port is set to `80`, but we would
like to use `8080`, so we'll again update `deploy/crds/helm.example.com_v1alpha1_nginx_cr.yaml`
by adding the service port override:

```yaml
apiVersion: helm.example.com/v1alpha1
kind: Nginx
metadata:
  name: example-nginx
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
kubectl create -f deploy/crds/helm.example.com_nginxes_crd.yaml
```

Once this is done, there are two ways to run the operator:

- As a pod inside a Kubernetes cluster
- As a go program outside the cluster using `operator-sdk`

### 1. Run as a pod inside a Kubernetes cluster

Running as a pod inside a Kubernetes cluster is preferred for production use.

Build the nginx-operator image and push it to a registry:

```sh
operator-sdk build quay.io/example/nginx-operator:v0.0.1
docker push quay.io/example/nginx-operator:v0.0.1
```

Kubernetes deployment manifests are generated in `deploy/operator.yaml`. The
deployment image in this file needs to be modified from the placeholder
`REPLACE_IMAGE` to the previous built image. To do this run:

```sh
sed -i 's|REPLACE_IMAGE|quay.io/example/nginx-operator:v0.0.1|g' deploy/operator.yaml
```

**Note**
If you are performing these steps on OSX, use the following `sed` command instead:

```sh
sed -i "" 's|REPLACE_IMAGE|quay.io/example/nginx-operator:v0.0.1|g' deploy/operator.yaml
```

Deploy the nginx-operator:

```sh
kubectl create -f deploy/service_account.yaml
kubectl create -f deploy/role.yaml
kubectl create -f deploy/role_binding.yaml
kubectl create -f deploy/operator.yaml
```

Verify that the nginx-operator is up and running:

```sh
$ kubectl get deployment
NAME                 DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
nginx-operator       1         1         1            1           1m
```

### 2. Run outside the cluster

This method is preferred during the development cycle to speed up deployment and testing.

It is important that the `chart` path referenced in `watches.yaml` exists
on your machine. By default, the `watches.yaml` file is scaffolded to work with
an operator image built with `operator-sdk build`. When developing and
testing your operator with `operator-sdk up local`, the SDK will look in your
local filesystem for this path. The SDK team recommends creating a symlink at
this location to point to your helm chart's path:

```sh
sudo mkdir -p /opt/helm/helm-charts
sudo ln -s $PWD/helm-charts/nginx /opt/helm/helm-charts/nginx
```

Run the operator locally with the default Kubernetes config file present at
`$HOME/.kube/config`:

```sh
$ operator-sdk up local
INFO[0000] Go Version: go1.10.3
INFO[0000] Go OS/Arch: linux/amd64
INFO[0000] operator-sdk Version: v0.1.1+git
```

Run the operator locally with a provided Kubernetes config file:

```sh
$ operator-sdk up local --kubeconfig=<path_to_config>
INFO[0000] Go Version: go1.10.3
INFO[0000] Go OS/Arch: linux/amd64
INFO[0000] operator-sdk Version: v0.2.0+git
```

## Deploy the Nginx custom resource

Apply the nginx CR that we modified earlier:

```sh
kubectl apply -f deploy/crds/helm.example.com_v1alpha1_nginx_cr.yaml
```

Ensure that the nginx-operator creates the deployment for the CR:

```sh
$ kubectl get deployment
NAME                                           DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
example-nginx-b9phnoz9spckcrua7ihrbkrt1        2         2         2            2           1m
```

Check the pods to confirm 2 replicas were created:

```sh
$ kubectl get pods
NAME                                                      READY     STATUS    RESTARTS   AGE
example-nginx-b9phnoz9spckcrua7ihrbkrt1-f8f9c875d-fjcr9   1/1       Running   0          1m
example-nginx-b9phnoz9spckcrua7ihrbkrt1-f8f9c875d-ljbzl   1/1       Running   0          1m
```

Check that the service port is set to `8080`:

```sh
$ kubectl get service
NAME                                      TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)    AGE
example-nginx-b9phnoz9spckcrua7ihrbkrt1   ClusterIP   10.96.26.3   <none>        8080/TCP   1m
```

### Update the replicaCount and remove the port

Change the `spec.replicaCount` field from 2 to 3, remove the `spec.service`
field, and apply the change:

```sh
$ cat deploy/crds/helm.example.com_v1alpha1_nginx_cr.yaml
apiVersion: "helm.example.com/v1alpha1"
kind: "Nginx"
metadata:
  name: "example-nginx"
spec:
  replicaCount: 3

$ kubectl apply -f deploy/crds/helm.example.com_v1alpha1_nginx_cr.yaml
```

Confirm that the operator changes the deployment size:

```sh
$ kubectl get deployment
NAME                                           DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
example-nginx-b9phnoz9spckcrua7ihrbkrt1        3         3         3            3           1m
```

Check that the service port is set to the default (`80`):

```sh
$ kubectl get service
NAME                                      TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)  AGE
example-nginx-b9phnoz9spckcrua7ihrbkrt1   ClusterIP   10.96.26.3   <none>        80/TCP   1m
```

### Cleanup

Clean up the resources:

```sh
kubectl delete -f deploy/crds/helm.example.com_v1alpha1_nginx_cr.yaml
kubectl delete -f deploy/operator.yaml
kubectl delete -f deploy/role_binding.yaml
kubectl delete -f deploy/role.yaml
kubectl delete -f deploy/service_account.yaml
kubectl delete -f deploy/crds/helm.example.com_nginxes_crd.yaml
```

[operator-scope]:./../operator-scope.md
[install-guide]: ../user/install-operator-sdk.md
[layout-doc]:./project_layout.md
[homebrew-tool]:https://brew.sh/
[git-tool]:https://git-scm.com/downloads
[go-tool]:https://golang.org/dl/
[docker-tool]:https://docs.docker.com/install/
[kubectl-tool]:https://kubernetes.io/docs/tasks/tools/install-kubectl/
[minikube-tool]:https://github.com/kubernetes/minikube#installation
[helm-charts]:https://helm.sh/docs/developing_charts/
[helm-values]:https://helm.sh/docs/using_helm/#customizing-the-chart-before-installing
[quay-link]:https://quay.io
