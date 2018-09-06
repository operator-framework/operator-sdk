# User Guide

This guide walks through an example of building a simple memcached-operator using tools and libraries provided by the Operator SDK.

## Prerequisites

- [dep][dep_tool] version v0.5.0+.
- [git][git_tool]
- [go][go_tool] version v1.10+.
- [docker][docker_tool] version 17.03+.
- [kubectl][kubectl_tool] version v1.9.0+.
- Access to a kubernetes v.1.9.0+ cluster.

**Note**: This guide uses [minikube][minikube_tool] version v0.25.0+ as the local kubernetes cluster and quay.io for the public registry.

## Install the Operator SDK CLI

The Operator SDK has a CLI tool that helps the developer to create, build, and deploy a new operator project.

Checkout the desired release tag and install the SDK CLI tool:

```sh
$ mkdir -p $GOPATH/src/github.com/operator-framework
$ cd $GOPATH/src/github.com/operator-framework
$ git clone https://github.com/operator-framework/operator-sdk
$ cd operator-sdk
$ git checkout master
$ make dep
$ make install
```

This installs the CLI binary `operator-sdk` at `$GOPATH/bin`.

## Create a new project

Use the CLI to create a new memcached-operator project:

```sh
$ mkdir -p $GOPATH/src/github.com/example-inc/
$ cd $GOPATH/src/github.com/example-inc/
$ operator-sdk new memcached-operator --api-version=cache.example.com/v1alpha1 --kind=Memcached
$ cd memcached-operator
```

This creates the memcached-operator project specifically for watching the Memcached resource with APIVersion `cache.example.com/v1apha1` and Kind `Memcached`.

To learn more about the project directory structure, see [project layout][layout_doc] doc.

## Customize the operator logic

For this example the memcached-operator will execute the following reconciliation logic for each `Memcached` CR:
- Create a memcached Deployment if it doesn't exist
- Ensure that the Deployment size is the same as specified by the `Memcached` CR spec
- Update the `Memcached` CR status with the names of the memcached pods

### Watch the Memcached CR

By default, the memcached-operator watches `Memcached` resource events as shown in `cmd/memcached-operator/main.go`.

```Go
func main() {
  sdk.Watch("cache.example.com/v1alpha1", "Memcached", "default", time.Duration(5)*time.Second)
  sdk.Handle(stub.NewHandler())
  sdk.Run(context.TODO())
}
```

#### Options
**Worker Count**
The number of concurrent informer workers can be configured with an additional Watch option. The default value is 1 if an argument is not given.
```Go
sdk.Watch("cache.example.com/v1alpha1", "Memcached", "default", time.Duration(5)*time.Second, sdk.WithNumWorkers(n))
```

**Label Selector**
Label selectors allow the watch to filter resources by kubernetes labels. It can be specified using the standard kubernetes label selector format:

https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors

```Go
sdk.Watch("cache.example.com/v1alpha1", "Memcached", "default", time.Duration(5)*time.Second, sdk.WithLabelSelector("app=myapp"))
```

### Define the Memcached spec and status

Modify the spec and status of the `Memcached` CR at `pkg/apis/cache/v1alpha1/types.go`:

```Go
type MemcachedSpec struct {
	// Size is the size of the memcached deployment
	Size int32 `json:"size"`
}
type MemcachedStatus struct {
	// Nodes are the names of the memcached pods
	Nodes []string `json:"nodes"`
}
```
Update the generated code for the CR:

```sh
$ operator-sdk generate k8s
```

### Define the Handler

The reconciliation loop for an event is defined in the `Handle()` function at `pkg/stub/handler.go`.

Replace the default handler with the reference [memcached handler][memcached_handler] implementation.

> Note: The provided handler implementation is only meant to demonstrate the use of the SDK APIs and is not representative of the best practices of a reconciliation loop.

### Build and run the operator

Before running the operator, Kubernetes needs to know about the new custom resource definition the operator will be watching.

Deploy the CRD:

```sh
$ kubectl create -f deploy/crd.yaml
```

Once this is done, there are two ways to run the operator:

- As pod inside Kubernetes cluster
- As go program outside cluster

#### 1. Run as pod inside a Kubernetes cluster

Run as pod inside a Kubernetes cluster is preferred for production use.

Build the memcached-operator image and push it to a registry:
```
$ operator-sdk build quay.io/example/memcached-operator:v0.0.1
$ sed -i 's|REPLACE_IMAGE|quay.io/example/memcached-operator:v0.0.1|g' deploy/operator.yaml
$ docker push quay.io/example/memcached-operator:v0.0.1
```

Kubernetes deployment manifests are generated in `deploy/operator.yaml`. The deployment image is set to the container image specified above.

Deploy the memcached-operator:

```sh
$ kubectl create -f deploy/rbac.yaml
$ kubectl create -f deploy/operator.yaml
```

Verify that the memcached-operator is up and running:

```sh
$ kubectl get deployment
NAME                     DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
memcached-operator       1         1         1            1           1m
```

#### 2. Run outside the cluster

This method is preferred during development cycle to deploy and test faster.

Run the operator locally with the default kubernetes config file present at `$HOME/.kube/config`:

```sh
$ operator-sdk up local
INFO[0000] Go Version: go1.10
INFO[0000] Go OS/Arch: darwin/amd64
INFO[0000] operator-sdk Version: 0.0.5+git
```

Run the operator locally with a provided kubernetes config file:

```sh
$ operator-sdk up local --kubeconfig=config
INFO[0000] Go Version: go1.10
INFO[0000] Go OS/Arch: darwin/amd64
INFO[0000] operator-sdk Version: 0.0.5+git
```

### Create a Memcached CR

Modify `deploy/cr.yaml` as shown and create a `Memcached` custom resource:

```sh
$ cat deploy/cr.yaml
apiVersion: "cache.example.com/v1alpha1"
kind: "Memcached"
metadata:
  name: "example-memcached"
spec:
  size: 3

$ kubectl apply -f deploy/cr.yaml
```

Ensure that the memcached-operator creates the deployment for the CR:

```sh
$ kubectl get deployment
NAME                     DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
memcached-operator       1         1         1            1           2m
example-memcached        3         3         3            3           1m
```

Check the pods and CR status to confirm the status is updated with the memcached pod names:

```sh
$ kubectl get pods
NAME                                  READY     STATUS    RESTARTS   AGE
example-memcached-6fd7c98d8-7dqdr     1/1       Running   0          1m
example-memcached-6fd7c98d8-g5k7v     1/1       Running   0          1m
example-memcached-6fd7c98d8-m7vn7     1/1       Running   0          1m
memcached-operator-7cc7cfdf86-vvjqk   1/1       Running   0          2m
```

```sh
$ kubectl get memcached/example-memcached -o yaml
apiVersion: cache.example.com/v1alpha1
kind: Memcached
metadata:
  clusterName: ""
  creationTimestamp: 2018-03-31T22:51:08Z
  generation: 0
  name: example-memcached
  namespace: default
  resourceVersion: "245453"
  selfLink: /apis/cache.example.com/v1alpha1/namespaces/default/memcacheds/example-memcached
  uid: 0026cc97-3536-11e8-bd83-0800274106a1
spec:
  size: 3
status:
  nodes:
  - example-memcached-6fd7c98d8-7dqdr
  - example-memcached-6fd7c98d8-g5k7v
  - example-memcached-6fd7c98d8-m7vn7
```

### Update the size

Change the `spec.size` field in the memcached CR from 3 to 4 and apply the change:

```sh
$ cat deploy/cr.yaml
apiVersion: "cache.example.com/v1alpha1"
kind: "Memcached"
metadata:
  name: "example-memcached"
spec:
  size: 4

$ kubectl apply -f deploy/cr.yaml
```

Confirm that the operator changes the deployment size:

```sh
$ kubectl get deployment
NAME                 DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
example-memcached    4         4         4            4           5m
```

### Cleanup

Clean up the resources:

```sh
$ kubectl delete -f deploy/cr.yaml
$ kubectl delete -f deploy/operator.yaml
```

[memcached_handler]: ../example/memcached-operator/handler.go.tmpl
[layout_doc]:./project_layout.md
[dep_tool]:https://golang.github.io/dep/docs/installation.html
[git_tool]:https://git-scm.com/downloads
[go_tool]:https://golang.org/dl/
[docker_tool]:https://docs.docker.com/install/
[kubectl_tool]:https://kubernetes.io/docs/tasks/tools/install-kubectl/
[minikube_tool]:https://github.com/kubernetes/minikube#installation
