# User Guide

This guide walks through an example of building a simple memcached-operator powered by Ansible using tools and libraries provided by the Operator SDK.

## Prerequisites

- [dep][dep_tool] version v0.5.0+.
- [git][git_tool]
- [go][go_tool] version v1.10+.
- [docker][docker_tool] version 17.03+.
- [kubectl][kubectl_tool] version v1.9.0+.
- [ansible][ansible_tool] version v2.6.0+
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

Use the CLI to create a new Ansible-based memcached-operator project:

```sh
$ mkdir -p $GOPATH/src/github.com/example-inc/
$ cd $GOPATH/src/github.com/example-inc/
$ operator-sdk new memcached-operator --api-version=cache.example.com/v1alpha1 --kind=Memcached --type=ansible
$ cd memcached-operator
```

This creates the memcached-operator project specifically for watching the Memcached resource with APIVersion `cache.example.com/v1apha1` and Kind `Memcached`.

To learn more about the project directory structure, see [project layout][layout_doc] doc.

## Customize the operator logic

For this example the memcached-operator will execute the following reconciliation logic for each `Memcached` CR:
- Create a memcached Deployment if it doesn't exist
- Ensure that the Deployment size is the same as specified by the `Memcached` CR spec

### Watch the Memcached CR

By default, the memcached-operator watches `Memcached` resource events as shown in `watches.yaml` and executes Ansible Role `Memached`:

```yaml
---
- version: v1alpha1
  group: cache.example.com
  kind: Memcached
```

#### Options
**Role**
Specifying a `role` option in `watches.yaml` will configure the operator to use this specified path when launching `ansible-runner` with an Ansible Role. This is the default.
```yaml
---
- version: v1alpha1
  group: cache.example.com
  kind: Memcached
  role: /opt/ansible/roles/Memcached
```

**Playbook**
Specifying a `playbook` option in `watches.yaml` will configure the operator to use this specified path when launching `ansible-runner` with an Ansible Playbook
```yaml
---
- version: v1alpha1
  group: cache.example.com
  kind: Memcached
  playbook: /opt/ansible/playbook.yaml
```

## Building the Memcached Ansible Role
### Define the Memcached spec

Defining the spec for an Ansible Operator can be done entirely in Ansible. The Ansible Operator will simply pass all key value pairs listed in the Custom Resource spec field along to Ansible as [variables](https://docs.ansible.com/ansible/2.5/user_guide/playbooks_variables.html#passing-variables-on-the-command-line). It is recommended that you perform some type validation in Ansible on the variables to ensure that your application is receiving expected input.

First, set a default in case the user doesn't set the `spec` field by modifying `roles/Memcached/defaults/main.yml`:
```yaml
size: 1
```

### Build and run the operator

Before running the operator, Kubernetes needs to know about the new custom resource definition the operator will be watching.

Deploy the CRD:

```sh
$ kubectl create -f deploy/crd.yaml
```

Once this is done, there are two ways to run the operator:

- As pod inside Kubernetes cluster
- As go program outside cluster using `operator-sdk`

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


## Advanced Topics
### Adding 3rd Party Resources To Your Operator
To add a resource to an operator, you must add it to a scheme. By creating an `AddToScheme` method or reusing one you can easily add a resource to your scheme. An [example][deployments_register] shows that you define a function and then use the [runtime][runtime_package] package to create a `SchemeBuilder`

#### Current Operator-SDK
You then need to tell the operators to use these functions to add the resources to its scheme. In operator-sdk you use [AddToSDKScheme][osdk_add_to_scheme] to add this.
Example of you main.go:
```go
import (
    ....
    appsv1 "k8s.io/api/apps/v1"
)

func main() {
    k8sutil.AddToSDKScheme(appsv1.AddToScheme)`
    sdk.Watch(appsv1.SchemeGroupVersion.String(), "Deployments", <namespace>, <resyncPeriod>)
}
```

#### Future with Controller Runtime
When using controller runtime, you will also need to tell its scheme about your resourece. In controller runtime to add to the scheme, you can get the managers [scheme][manager_scheme].  If you would like to see what kubebuilder generates to add the resoureces to the [scheme][simple_resource].
Example:
```go
import (
    ....
    appsv1 "k8s.io/api/apps/v1"
)

func main() {
    ....
    if err := appsv1.AddToScheme(mgr.GetScheme()); err != nil {
        log.Fatal(err)
    }
    ....
}
```

[memcached_handler]: ../example/memcached-operator/handler.go.tmpl
[layout_doc]:./project_layout.md
[dep_tool]:https://golang.github.io/dep/docs/installation.html
[git_tool]:https://git-scm.com/downloads
[go_tool]:https://golang.org/dl/
[docker_tool]:https://docs.docker.com/install/
[kubectl_tool]:https://kubernetes.io/docs/tasks/tools/install-kubectl/
[minikube_tool]:https://github.com/kubernetes/minikube#installation
[manager_scheme]: https://github.com/kubernetes-sigs/controller-runtime/blob/master/pkg/manager/manager.go#L61
[simple_resource]: https://book.kubebuilder.io/basics/simple_resource.html
[deployments_register]: https://github.com/kubernetes/api/blob/master/apps/v1/register.go#L41
[runtime_package]: https://godoc.org/k8s.io/apimachinery/pkg/runtime
[osdk_add_to_scheme]: https://github.com/operator-framework/operator-sdk/blob/4179b6ac459b2b0cb04ab3a1b438c280bd28d1a5/pkg/util/k8sutil/k8sutil.go#L67
