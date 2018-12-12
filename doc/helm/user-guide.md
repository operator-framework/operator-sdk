# User Guide

This guide walks through an example of building a simple nginx-operator
powered by Helm using tools and libraries provided by the Operator SDK.

## Prerequisites

- [git][git_tool]
- [docker][docker_tool] version 17.03+.
- [kubectl][kubectl_tool] version v1.9.0+.
- [dep][dep_tool] version v0.5.0+. (Optional if you aren't installing from source)
- [go][go_tool] version v1.10+. (Optional if you aren't installing from source)
- Access to a kubernetes v.1.9.0+ cluster.

**Note**: This guide uses [minikube][minikube_tool] version v0.25.0+ as the
local kubernetes cluster and quay.io for the public registry.

## Install the Operator SDK CLI

The Operator SDK has a CLI tool that helps the developer to create, build, and
deploy a new operator project.

Checkout the desired release tag and install the SDK CLI tool:

```sh
mkdir -p $GOPATH/src/github.com/operator-framework
cd $GOPATH/src/github.com/operator-framework
git clone https://github.com/operator-framework/operator-sdk
cd operator-sdk
git checkout master
make dep
make install
```

This installs the CLI binary `operator-sdk` at `$GOPATH/bin`.

## Create a new project

Use the CLI to create a new Helm-based nginx-operator project:

```sh
operator-sdk new nginx-operator --api-version=example.com/v1alpha1 --kind=Nginx --type=helm
cd nginx-operator
```

This creates the nginx-operator project specifically for watching the
Nginx resource with APIVersion `example.com/v1apha1` and Kind
`Nginx`.

To learn more about the project directory structure, see the 
[project layout][layout_doc] doc.

### Operator scope

A namespace-scoped operator (the default) watches and manages resources in a single namespace, whereas a cluster-scoped operator watches and manages resources cluster-wide. Namespace-scoped operators are preferred because of their flexibility. They enable decoupled upgrades, namespace isolation for failures and monitoring, and differing API definitions. However, there are use cases where a cluster-scoped operator may make sense. For example, the [cert-manager](https://github.com/jetstack/cert-manager) operator is often deployed with cluster-scoped permissions and watches so that it can manage issuing certificates for an entire cluster.

If you'd like to create your nginx-operator project to be cluster-scoped use the following `operator-sdk new` command instead:

```sh
operator-sdk new nginx-operator --cluster-scoped --api-version=example.com/v1alpha1 --kind=Nginx --type=helm
```

Using `--cluster-scoped` will scaffold the new operator with the following modifications:
* `deploy/operator.yaml` - Set `WATCH_NAMESPACE=""` instead of setting it to the pod's namespace
* `deploy/role.yaml` - Use `ClusterRole` instead of `Role`
* `deploy/role_binding.yaml`:
  * Use `ClusterRoleBinding` instead of `RoleBinding`
  * Set the subject namespace to `REPLACE_NAMESPACE`. This must be changed to the namespace in which the operator is deployed.

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
  group: example.com
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
the [Helm Chart developer documentation][helm_charts].

### Understanding the Nginx CR spec

Helm uses a concept called [values][helm_values] to provide customizations
to a Helm chart's defaults, which are defined in the Helm chart's `values.yaml`
file.

Overriding these defaults is a simple as setting the desired values in the CR
spec. Let's use the number of replicas as an example.

First, inspecting `helm-charts/nginx/values.yaml`, we see that the chart has a
value called `replicaCount` and it is set to `1` by default. If we want to have
2 nginx instances in our deployment, we would need to make sure our CR spec
contained `replicaCount: 2`.

Update `deploy/crds/example_v1alpha1_nginx_cr.yaml` to look like the following:

```yaml
apiVersion: example.com/v1alpha1
kind: Nginx
metadata:
  name: example-nginx
spec:
  replicaCount: 2
```

Similarly, we see that the default service port is set to `80`, but we would
like to use `8080`, so we'll again update `deploy/crds/example_v1alpha1_nginx_cr.yaml`
by adding the service port override:

```yaml
apiVersion: example.com/v1alpha1
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
kubectl create -f deploy/crds/example_v1alpha1_nginx_crd.yaml
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

If you created your operator using `--cluster-scoped=true`, update the service account namespace in the generated `ClusterRoleBinding` to match where you are deploying your operator.

```sh
export OPERATOR_NAMESPACE=$(kubectl config view --minify -o jsonpath='{.contexts[0].context.namespace}')
sed -i "s|REPLACE_NAMESPACE|$OPERATOR_NAMESPACE|g" deploy/role_binding.yaml
```

**Note**  
If you are performing these steps on OSX, use the following commands instead:

```sh
sed -i "" 's|REPLACE_IMAGE|quay.io/example/nginx-operator:v0.0.1|g' deploy/operator.yaml
sed -i "" "s|REPLACE_NAMESPACE|$OPERATOR_NAMESPACE|g" deploy/role_binding.yaml
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

Run the operator locally with the default kubernetes config file present at
`$HOME/.kube/config`:

```sh
$ operator-sdk up local
INFO[0000] Go Version: go1.10.3
INFO[0000] Go OS/Arch: linux/amd64
INFO[0000] operator-sdk Version: v0.1.1+git
```

Run the operator locally with a provided kubernetes config file:

```sh
$ operator-sdk up local --kubeconfig=<path_to_config>
INFO[0000] Go Version: go1.10.3
INFO[0000] Go OS/Arch: linux/amd64
INFO[0000] operator-sdk Version: v0.2.0+git
```

## Deploy the Nginx custom resource

Apply the nginx CR that we modified earlier:

```sh
kubectl apply -f deploy/crds/example_v1alpha1_nginx_cr.yaml
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
$ cat deploy/crds/example_v1alpha1_nginx_cr.yaml
apiVersion: "example.com/v1alpha1"
kind: "Nginx"
metadata:
  name: "example-nginx"
spec:
  replicaCount: 3

$ kubectl apply -f deploy/crds/example_v1alpha1_nginx_cr.yaml
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
kubectl delete -f deploy/crds/example_v1alpha1_nginx_cr.yaml
kubectl delete -f deploy/operator.yaml
kubectl delete -f deploy/role_binding.yaml
kubectl delete -f deploy/role.yaml
kubectl delete -f deploy/service_account.yaml
kubectl delete -f deploy/crds/example_v1alpha1_nginx_cr.yaml
```

[layout_doc]:./project_layout.md
[dep_tool]:https://golang.github.io/dep/docs/installation.html
[git_tool]:https://git-scm.com/downloads
[go_tool]:https://golang.org/dl/
[docker_tool]:https://docs.docker.com/install/
[kubectl_tool]:https://kubernetes.io/docs/tasks/tools/install-kubectl/
[minikube_tool]:https://github.com/kubernetes/minikube#installation
[helm_charts]:https://docs.helm.sh/developing_charts/
[helm_values]:https://docs.helm.sh/using_helm/#customizing-the-chart-before-installing
