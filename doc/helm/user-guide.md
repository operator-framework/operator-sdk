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
Nginx resource with APIVersion `example.com/v1alpha1` and Kind
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

If `--helm-chart-version` is not set, the SDK will fetch the latest available version of the helm chart. Otherwise, it will fetch the specified version. The option `--helm-chart-version` is not used when `--helm-chart` itself refers to a specific version, for example when it is a local path or a URL.

**Note:** For more details and examples see the [Helm CLI reference doc][helm-reference-cli-doc].


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
  group: example.com
  kind: Nginx
  chart: helm-charts/nginx
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

Update `deploy/crds/example.com_v1alpha1_nginx_cr.yaml` to look like the following:

```yaml
apiVersion: example.com/v1alpha1
kind: Nginx
metadata:
  name: example-nginx
spec:
  replicaCount: 2
```

Similarly, we see that the default service port is set to `80`, but we would
like to use `8080`, so we'll again update `deploy/crds/example.com_v1alpha1_nginx_cr.yaml`
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
kubectl create -f deploy/crds/example.com_nginxes_crd.yaml
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

Run the operator locally with the default Kubernetes config file present at
`$HOME/.kube/config`:

```sh
$ operator-sdk run --local
INFO[0000] Go Version: go1.10.3
INFO[0000] Go OS/Arch: linux/amd64
INFO[0000] operator-sdk Version: v0.1.1+git
```

Run the operator locally with a provided Kubernetes config file:

```sh
$ operator-sdk run --local --kubeconfig=<path_to_config>
INFO[0000] Go Version: go1.10.3
INFO[0000] Go OS/Arch: linux/amd64
INFO[0000] operator-sdk Version: v0.2.0+git
```

## Deploy the Nginx custom resource

Apply the nginx CR that we modified earlier:

```sh
kubectl apply -f deploy/crds/example.com_v1alpha1_nginx_cr.yaml
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
$ cat deploy/crds/example.com_v1alpha1_nginx_cr.yaml
apiVersion: "example.com/v1alpha1"
kind: "Nginx"
metadata:
  name: "example-nginx"
spec:
  replicaCount: 3

$ kubectl apply -f deploy/crds/example.com_v1alpha1_nginx_cr.yaml
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
kubectl delete -f deploy/crds/example.com_v1alpha1_nginx_cr.yaml
kubectl delete -f deploy/operator.yaml
kubectl delete -f deploy/role_binding.yaml
kubectl delete -f deploy/role.yaml
kubectl delete -f deploy/service_account.yaml
kubectl delete -f deploy/crds/example.com_nginxes_crd.yaml
```

## Advanced features

## Passing environment variables to the Helm chart

Sometimes it is useful to pass down environment variables from the Operators `Deployment`
all the way to the helm charts templates. This allows the Operator to be configured at a global
level at runtime. This is new compared to dealing with the helm CLI
as they usually don't have access to any environment variables in the context of Tiller (helm v2)
or the helm binary (helm v3) for security reasons.

With the helm Operator this becomes possible by override values. This enforces that certain
template values provided by the chart's default `values.yaml` or by a CR spec are always set
when rendering the chart. If the value is set by a CR it gets overridden by the global override value.
The override value can be static but can also refer to an environment variable. To pass down environment
variables to the chart override values is currently the only way.

An example use case of this is when your helm chart references container images by chart variables,
which is a good practice.
If your Operator is deployed in a disconnected environment (no network access to the default images
location) you can use this mechanism to set them globally at the Operator level using environment variables
versus individually per CR / chart release.

> Note that it is strongly recommended to reference container images in your chart by helm variables
> and then also associate these with an environment variable of your Operator like shown below.
> This allows your Operator to be mirrored for offline usage when packaged for OLM.

To configure your operator with override values, add an `overrideValues` map to your
`watches.yaml` file for the GVK and chart you need to override. For example, to change
the repository used by the nginx chart, you would update your `watches.yaml` to the
following:

```yaml
---
- version: v1alpha1
  group: example.com
  kind: Nginx
  chart: helm-charts/nginx
  overrideValues:
    image.repository: quay.io/mycustomrepo
```

By setting `image.repository` to `quay.io/mycustomrepo` you are ensuring that
`quay.io/mycustomrepo` will always be used instead of the chart's default repository
(`nginx`). If the CR attempts to set this value, it will be ignored.

It is now possible to reference environment variables in the `overrideValues` section:

```yaml
  overrideValues:
    image.repository: $IMAGE_REPOSITORY # or ${IMAGE_REPOSITORY}
```

By using an environment variable reference in `overrideValues` you enable these override
values to be set at runtime by configuring the environment variable on the
operator deployment. For example, in `deploy/operator.yaml` you could add the
following snippet to the container spec:

```yaml
env:
  - name: IMAGE_REPOSITORY
    value: quay.io/mycustomrepo
```

If an environment variable reference is listed in `overrideValues`, but is not present
in the environment when the operator runs, it will resolve to an empty string and
override all other values. Therefore, these environment variables should _always_ be
set. It is suggested to update the Dockerfile to set these environment variables to
the same defaults that are defined by the chart.

To warn users that their CR settings may be ignored, the Helm operator creates events on
the CR that include the name and value of each overridden value. For example:

```
Events:
  Type     Reason               Age   From              Message
  ----     ------               ----  ----              -------
  Warning  OverrideValuesInUse  1m    nginx-controller  Chart value "image.repository" overridden to "quay.io/mycustomrepo" by operator's watches.yaml
```


## Changing the concurrent worker count

Depending on the number of CRs of the same type, a single reconciling worker may have issues keeping up. You can increase the number of workers by passing `--max-workers <number of workers>`.

For example:

```sh
$ operator-sdk exec-entrypoint helm --max-workers 10
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
[helm-charts]:https://helm.sh/docs/topics/charts/
[helm-values]:https://helm.sh/docs/intro/using_helm/#customizing-the-chart-before-installing
[quay-link]:https://quay.io
[helm-reference-cli-doc]:./../sdk-cli-reference.md#helm-project
