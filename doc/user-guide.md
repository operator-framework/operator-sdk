# User Guide

This guide walks through an example of building a simple memcached-operator using the operator-sdk
CLI tool and controller-runtime library API. To learn how to use Ansible or Helm to create an
operator, see the [Ansible Operator User Guide][ansible_user_guide] or the [Helm Operator User
Guide][helm_user_guide]. The rest of this document will show how to program an operator in Go.

## Prerequisites

- [dep][dep_tool] version v0.5.0+.
- [git][git_tool]
- [go][go_tool] version v1.10+.
- [docker][docker_tool] version 17.03+.
- [kubectl][kubectl_tool] version v1.11.0+.
- Access to a kubernetes v.1.11.0+ cluster.

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
$ operator-sdk new memcached-operator
$ cd memcached-operator
```

To learn about the project directory structure, see [project layout][layout_doc] doc.

#### Operator scope

A namespace-scoped operator (the default) watches and manages resources in a single namespace, whereas a cluster-scoped operator watches and manages resources cluster-wide. Namespace-scoped operators are preferred because of their flexibility. They enable decoupled upgrades, namespace isolation for failures and monitoring, and differing API definitions. However, there are use cases where a cluster-scoped operator may make sense. For example, the [cert-manager](https://github.com/jetstack/cert-manager) operator is often deployed with cluster-scoped permissions and watches so that it can manage issuing certificates for an entire cluster.

If you'd like to create your memcached-operator project to be cluster-scoped use the following `operator-sdk new` command instead:
```
$ operator-sdk new memcached-operator --cluster-scoped
```

Using `--cluster-scoped` will scaffold the new operator with the following modifications:
* `deploy/operator.yaml` - Set `WATCH_NAMESPACE=""` instead of setting it to the pod's namespace
* `deploy/role.yaml` - Use `ClusterRole` instead of `Role`
* `deploy/role_binding.yaml`:
  * Use `ClusterRoleBinding` instead of `RoleBinding`
  * Set the subject namespace to `REPLACE_NAMESPACE`. This must be changed to the namespace in which the operator is deployed.

### Manager
The main program for the operator `cmd/manager/main.go` initializes and runs the [Manager][manager_go_doc].

The Manager will automatically register the scheme for all custom resources defined under `pkg/apis/...` and run all controllers under `pkg/controller/...`.

The Manager can restrict the namespace that all controllers will watch for resources:
```Go
mgr, err := manager.New(cfg, manager.Options{Namespace: namespace})
```
By default this will be the namespace that the operator is running in. To watch all namespaces leave the namespace option empty:
```Go
mgr, err := manager.New(cfg, manager.Options{Namespace: ""})
```

## Add a new Custom Resource Definition

Add a new Custom Resource Definition(CRD) API called Memcached, with APIVersion `cache.example.com/v1apha1` and Kind `Memcached`.

```sh
$ operator-sdk add api --api-version=cache.example.com/v1alpha1 --kind=Memcached
```

This will scaffold the Memcached resource API under `pkg/apis/cache/v1alpha1/...`.

### Define the spec and status

Modify the spec and status of the `Memcached` Custom Resource(CR) at `pkg/apis/cache/v1alpha1/memcached_types.go`:

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

After modifying the `*_types.go` file always run the following command to update the generated code for that resource type:

```sh
$ operator-sdk generate k8s
```

## Add a new Controller

Add a new [Controller][controller-go-doc] to the project that will watch and reconcile the Memcached resource:

```sh
$ operator-sdk add controller --api-version=cache.example.com/v1alpha1 --kind=Memcached
```

This will scaffold a new Controller implementation under `pkg/controller/memcached/...`.

For this example replace the generated Controller file `pkg/controller/memcached/memcached_controller.go` with the example [`memcached_controller.go`][memcached_controller] implementation.

The example Controller executes the following reconciliation logic for each `Memcached` CR:
- Create a memcached Deployment if it doesn't exist
- Ensure that the Deployment size is the same as specified by the `Memcached` CR spec
- Update the `Memcached` CR status using the status writer with the names of the memcached pods

The next two subsections explain how the Controller watches resources and how the reconcile loop is triggered. Skip to the [Build](#build-and-run-the-operator) section to see how to build and run the operator.

### Resources watched by the Controller

Inspect the Controller implementation at `pkg/controller/memcached/memcached_controller.go` to see how the Controller watches resources.

The first watch is for the Memcached type as the primary resource. For each Add/Update/Delete event the reconcile loop will be sent a reconcile `Request` (a namespace/name key) for that Memcached object:

```Go
err := c.Watch(
  &source.Kind{Type: &cachev1alpha1.Memcached{}}, &handler.EnqueueRequestForObject{})
```

The next watch is for Deployments but the event handler will map each event to a reconcile `Request` for the owner of the Deployment. Which in this case is the Memcached object for which the Deployment was created. This allows the controller to watch Deployments as a secondary resource.

```Go
err := c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &cachev1alpha1.Memcached{},
	})
```

**// TODO:** Doc on eventhandler, arbitrary mapping between watched and reconciled resource.

**// TODO:** Doc on configuring a Controller: number of workers, predicates, watching channels,

### Reconcile loop

Every Controller has a Reconciler object with a `Reconcile()` method that implements the reconcile loop. The reconcile loop is passed the [`Request`][request-go-doc] argument which is a Namespace/Name key used to lookup the primary resource object, Memcached, from the cache:

```Go
func (r *ReconcileMemcached) Reconcile(request reconcile.Request) (reconcile.Result, error) {
  // Lookup the Memcached instance for this reconcile request
  memcached := &cachev1alpha1.Memcached{}
  err := r.client.Get(context.TODO(), request.NamespacedName, memcached)
  ...
}  
```

Based on the return values, [`Result`][result_go_doc] and error, the `Request` may be requeued and the reconcile loop may be triggered again:

```Go
// Reconcile successful - don't requeue
return reconcile.Result{}, nil
// Reconcile failed due to error - requeue
return reconcile.Result{}, err
// Requeue for any reason other than error
return reconcile.Result{Requeue: true}, nil
```

You can set the `Result.RequeueAfter` to requeue the `Request` after a grace period as well:
```Go
import "time"

// Reconcile for any reason than error after 5 seconds
return reconcile.Result{RequeueAfter: time.Second*5}, nil
```

**Note:** Returning `Result` with `RequeueAfter` set is how you can periodically reconcile a CR.

For a guide on Reconcilers, Clients, and interacting with resource Events, see the [Client API doc][doc_client_api].

## Build and run the operator

Before running the operator, the CRD must be registered with the Kubernetes apiserver:

```sh
$ kubectl create -f deploy/crds/cache_v1alpha1_memcached_crd.yaml
```

Once this is done, there are two ways to run the operator:

- As a Deployment inside a Kubernetes cluster
- As Go program outside a cluster

### 1. Run as a Deployment inside the cluster

Build the memcached-operator image and push it to a registry:
```
$ operator-sdk build quay.io/example/memcached-operator:v0.0.1
$ sed -i 's|REPLACE_IMAGE|quay.io/example/memcached-operator:v0.0.1|g' deploy/operator.yaml
$ docker push quay.io/example/memcached-operator:v0.0.1
```

If you created your operator using `--cluster-scoped=true`, update the service account namespace in the generated `ClusterRoleBinding` to match where you are deploying your operator.
```
$ export OPERATOR_NAMESPACE=$(kubectl config view --minify -o jsonpath='{.contexts[0].context.namespace}')
$ sed -i "s|REPLACE_NAMESPACE|$OPERATOR_NAMESPACE|g" deploy/role_binding.yaml
```

**Note**  
If you are performing these steps on OSX, use the following commands instead:
```
$ sed -i "" 's|REPLACE_IMAGE|quay.io/example/memcached-operator:v0.0.1|g' deploy/operator.yaml
$ sed -i "" "s|REPLACE_NAMESPACE|$OPERATOR_NAMESPACE|g" deploy/role_binding.yaml
```

The Deployment manifest is generated at `deploy/operator.yaml`. Be sure to update the deployment image as shown above since the default is just a placeholder.

Setup RBAC and deploy the memcached-operator:

```sh
$ kubectl create -f deploy/service_account.yaml
$ kubectl create -f deploy/role.yaml
$ kubectl create -f deploy/role_binding.yaml
$ kubectl create -f deploy/operator.yaml
```

Verify that the memcached-operator is up and running:

```sh
$ kubectl get deployment
NAME                     DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
memcached-operator       1         1         1            1           1m
```

### 2. Run locally outside the cluster

This method is preferred during development cycle to deploy and test faster.

Set the name of the operator in an environment variable:

```sh
export OPERATOR_NAME=memcached-operator
```

Run the operator locally with the default kubernetes config file present at `$HOME/.kube/config`:

```sh
$ operator-sdk up local --namespace=default
2018/09/30 23:10:11 Go Version: go1.10.2
2018/09/30 23:10:11 Go OS/Arch: darwin/amd64
2018/09/30 23:10:11 operator-sdk Version: 0.0.6+git
2018/09/30 23:10:12 Registering Components.
2018/09/30 23:10:12 Starting the Cmd.
```

You can use a specific kubeconfig via the flag `--kubeconfig=<path/to/kubeconfig>`.

## Create a Memcached CR

Create the example `Memcached` CR that was generated at `deploy/crds/cache_v1alpha1_memcached_cr.yaml`:

```sh
$ cat deploy/crds/cache_v1alpha1_memcached_cr.yaml
apiVersion: "cache.example.com/v1alpha1"
kind: "Memcached"
metadata:
  name: "example-memcached"
spec:
  size: 3

$ kubectl apply -f deploy/crds/cache_v1alpha1_memcached_cr.yaml
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
$ cat deploy/crds/cache_v1alpha1_memcached_cr.yaml
apiVersion: "cache.example.com/v1alpha1"
kind: "Memcached"
metadata:
  name: "example-memcached"
spec:
  size: 4

$ kubectl apply -f deploy/crds/cache_v1alpha1_memcached_cr.yaml
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
$ kubectl delete -f deploy/crds/cache_v1alpha1_memcached_cr.yaml
$ kubectl delete -f deploy/operator.yaml
$ kubectl delete -f deploy/role_binding.yaml
$ kubectl delete -f deploy/role.yaml
$ kubectl delete -f deploy/service_account.yaml
```

## Advanced Topics

### Adding 3rd Party Resources To Your Operator

The operator's Manager supports the Core Kubernetes resource types as found in the client-go [scheme][scheme_package] package and will also register the schemes of all custom resource types defined in your project under `pkg/apis`.
```Go
import (
  "github.com/example-inc/memcached-operator/pkg/apis"
  ...
)
// Setup Scheme for all resources
if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
  log.Fatal(err)
}
```

To add a 3rd party resource to an operator, you must add it to the Manager's scheme. By creating an `AddToScheme` method or reusing one you can easily add a resource to your scheme. An [example][deployments_register] shows that you define a function and then use the [runtime][runtime_package] package to create a `SchemeBuilder`.

#### Register with the Manager's scheme

Call the `AddToScheme()` function for your 3rd party resource and pass it the Manager's scheme via `mgr.GetScheme()`.

Example:
```go
import (
    ....
    routev1 "github.com/openshift/api/route/v1"
)

func main() {
    ....
    if err := routev1.AddToScheme(mgr.GetScheme()); err != nil {
        log.Fatal(err)
    }
    ....
}
```

After adding new import paths to your operator project, run `dep ensure` in the root of your project directory to fulfill these dependencies.

[memcached_handler]: ../example/memcached-operator/handler.go.tmpl
[memcached_controller]: ../example/memcached-operator/memcached_controller.go.tmpl
[layout_doc]:./project_layout.md
[ansible_user_guide]:./ansible/user-guide.md
[helm_user_guide]:./helm/user-guide.md
[dep_tool]:https://golang.github.io/dep/docs/installation.html
[git_tool]:https://git-scm.com/downloads
[go_tool]:https://golang.org/dl/
[docker_tool]:https://docs.docker.com/install/
[kubectl_tool]:https://kubernetes.io/docs/tasks/tools/install-kubectl/
[minikube_tool]:https://github.com/kubernetes/minikube#installation
[scheme_package]:https://github.com/kubernetes/client-go/blob/master/kubernetes/scheme/register.go
[deployments_register]: https://github.com/kubernetes/api/blob/master/apps/v1/register.go#L41
[doc_client_api]:./user/client.md
[runtime_package]: https://godoc.org/k8s.io/apimachinery/pkg/runtime
[manager_go_doc]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/manager#Manager
[controller-go-doc]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg#hdr-Controller
[request-go-doc]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/reconcile#Request
[result_go_doc]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/reconcile#Result
