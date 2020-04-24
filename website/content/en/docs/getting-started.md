---
title:  Getting Started with Operator SDK
linkTitle: Getting Started
weight: 2
---
{{% alert title="Warning" color="warning" %}}
These pages are under construction. Please continue to use the [docs in
tree](https://github.com/operator-framework/operator-sdk/tree/master/doc)
for now.
{{% /alert %}}

# Getting Started

- [Overview](#overview)
- [Build an operator using the Operator SDK](#build-an-operator-using-the-operator-sdk)
  - [Create a new project](#create-a-new-project)
  - [Manager](#manager)
- [Add a new Custom Resource Definition](#add-a-new-custom-resource-definition)
  - [Define the Memcached spec and status](#define-the-memcached-spec-and-status)
- [Add a new Controller](#add-a-new-controller)
  - [Resources watched by the Controller](#resources-watched-by-the-controller)
  - [Reconcile loop](#reconcile-loop)
- [Build and run the operator](#build-and-run-the-operator)
  - [1. Run as a Deployment inside the cluster](#1-run-as-a-deployment-inside-the-cluster)
  - [2. Run locally outside the cluster](#2-run-locally-outside-the-cluster)
- [Create a Memcached CR](#create-a-memcached-cr)
  - [Update the size](#update-the-size)
  - [Cleanup](#cleanup)
- [Reference implementation](#reference-implementation)
- [Manage the operator using the Operator Lifecycle Manager](#manage-the-operator-using-the-operator-lifecycle-manager)
  - [Generate an operator manifest](#generate-an-operator-manifest)
  - [Testing locally](#testing-locally)
  - [Promoting operator standards](#promoting-operator-standards)
- [Conclusion](#conclusion)

## Overview

The [Operator Framework][org_operator_framework] ([intro blog post][site_blog_post]) is an open source toolkit to manage Kubernetes native applications, called operators, in an effective, automated, and scalable way. Operators take advantage of Kubernetes's extensibility to deliver the automation advantages of cloud services like provisioning, scaling, and backup/restore while being able to run anywhere that Kubernetes can run.

This guide shows how to build a simple [memcached][site_memcached] operator and how to manage its lifecycle from install to update to a new version. For that, we will use two center pieces of the framework:

* **Operator SDK**: Allows your developers to build an operator based on your expertise without requiring knowledge of Kubernetes API complexities.
* **Operator Lifecycle Manager**: Helps you to install, update, and generally manage the lifecycle of all of the operators (and their associated services) running across your clusters.

## Build an operator using the Operator SDK

**BEFORE YOU BEGIN:** links to the Operator SDK repo in this document are pinned to the `master` branch. Make sure you update the link such that it points to the correct Operator SDK repo version, which should match this repo's version or the `operator-sdk version` being used. For example, if you are using `operator-sdk` v0.12.0, update all links from this repo to the SDK repo with `master -> v0.12.0`. Otherwise you may see incorrect information.

The Operator SDK makes it easier to build Kubernetes native applications, a process that can require deep, application-specific operational knowledge. The SDK not only lowers that barrier, but it also helps reduce the amount of boilerplate code needed for many common management capabilities, such as metering or monitoring.

This section walks through an example of building a simple memcached operator using tools and libraries provided by the Operator SDK. This walkthrough is not exhaustive; for an in-depth explanation of these steps, see the SDK's [user guide][doc_sdk_user_guide].

**Requirements**: Please make sure that the Operator SDK is [installed][doc_sdk_install_instr] on the development machine. Additionally, the Operator Lifecycle Manager must be [installed][doc_olm_install_instr] in the cluster (1.8 or above to support the apps/v1beta2 API group) before running this guide.

### Create a new project

1. Use the CLI to create a new `memcached-operator` project:


```sh
$ mkdir -p $GOPATH/src/github.com/example-inc/
$ cd $GOPATH/src/github.com/example-inc/
$ export GO111MODULE=on
$ operator-sdk new memcached-operator
$ cd memcached-operator
```

This creates the `memcached-operator` project.

2. Install dependencies by running `go mod tidy`

**NOTE:** Learn more about the project directory structure from the SDK [project layout][layout_doc] documentation.

### Manager

The main program for the operator `cmd/manager/main.go` initializes and runs the [Manager][manager_go_doc].

The Manager will automatically register the scheme for all custom resources defined under `pkg/apis/...` and run all controllers under `pkg/controller/...`.

The Manager can restrict the namespace that all controllers will watch for resources:

```Go
mgr, err := manager.New(cfg, manager.Options{
	Namespace: namespace,
})
```

By default this will be the namespace that the operator is running in. To watch all namespaces leave the namespace option empty:

```Go
mgr, err := manager.New(cfg, manager.Options{
	Namespace: "",
})
```

## Add a new Custom Resource Definition

Add a new Custom Resource Definition (CRD) API called `Memcached`, with APIVersion `cache.example.com/v1alpha1` and Kind `Memcached`.

```sh
$ operator-sdk add api --api-version=cache.example.com/v1alpha1 --kind=Memcached
```

This will scaffold the `Memcached` resource API under `pkg/apis/cache/v1alpha1/...`.

### Define the Memcached spec and status

Modify the spec and status of the `Memcached` Custom Resource (CR) at `pkg/apis/cache/v1alpha1/memcached_types.go`:

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

Also run the following command in order to automatically generate the CRDs:

```sh
$ operator-sdk generate crds
```

You can see the changes applied in `deploy/crds/cache.example.com_memcacheds_crd.yaml`

## Add a new Controller

Add a new [Controller][controller_go_doc] to the project that will watch and reconcile the `Memcached` resource:

```sh
$ operator-sdk add controller --api-version=cache.example.com/v1alpha1 --kind=Memcached
```

This will scaffold a new Controller implementation under `pkg/controller/memcached/...`.

For this example replace the generated Controller file `pkg/controller/memcached/memcached_controller.go` with the example [`memcached_controller.go`][memcached_controller] implementation.

The example Controller executes the following reconciliation logic for each `Memcached` CR:

* Create a memcached Deployment if it doesn't exist
* Ensure that the Deployment size is the same as specified by the `Memcached` CR spec
* Update the `Memcached` CR status with the names of the memcached pods

The next two subsections explain how the Controller watches resources and how the reconcile loop is triggered. Skip to the [Build](#build-and-run-the-operator) section to see how to build and run the operator.

### Resources watched by the Controller

Inspect the Controller implementation at `pkg/controller/memcached/memcached_controller.go` to see how the Controller watches resources.

The first watch is for the `Memcached` type as the primary resource. For each Add/Update/Delete event the reconcile loop will be sent a reconcile `Request` (a namespace/name key) for that `Memcached` object:

```Go
err := c.Watch(
    &source.Kind{Type: &cachev1alpha1.Memcached{}},
    &handler.EnqueueRequestForObject{},
  )
```

The next watch is for Deployments but the event handler will map each event to a reconcile `Request` for the owner of the Deployment. Which in this case is the `Memcached` object for which the Deployment was created. This allows the controller to watch Deployments as a secondary resource.

```Go
err := c.Watch(
    &source.Kind{Type: &appsv1.Deployment{}},
    &handler.EnqueueRequestForOwner{
        IsController: true,
        OwnerType:    &cachev1alpha1.Memcached{}},
    )
```

### Reconcile loop

Every Controller has a Reconciler object with a `Reconcile()` method that implements the reconcile loop. The reconcile loop is passed the [`Request`][request_go_doc] argument which is a Namespace/Name key used to lookup the primary resource object, `Memcached`, from the cache:

```Go
func (r *ReconcileMemcached) Reconcile(request reconcile.Request) (reconcile.Result, error) {
    // Lookup the Memcached instance for this reconcile request
    memcached := &cachev1alpha1.Memcached{}
    err := r.client.Get(context.TODO(), request.NamespacedName, memcached)
    ...
}  
```

For a guide on Reconcilers, Clients, and interacting with resource Events, see the [Client API doc][doc_client_api].

## Build and run the operator

Before running the operator, the CRD must be registered with the Kubernetes apiserver:

```sh
$ kubectl create -f deploy/crds/cache.example.com_memcacheds_crd.yaml
```

Once this is done, there are two ways to run the operator:

* As a Deployment inside a Kubernetes cluster
* As Go program outside a cluster

### 1. Run as a Deployment inside the cluster

Build the memcached-operator image and push it to your registry. The following example uses https://quay.io as the registry.

```sh
$ operator-sdk build quay.io/<user>/memcached-operator:v0.0.1
$ sed -i 's|REPLACE_IMAGE|quay.io/<user>/memcached-operator:v0.0.1|g' deploy/operator.yaml
$ docker push quay.io/<user>/memcached-operator:v0.0.1
```

**Note**
If you are performing these steps on OSX, use the following `sed` command instead:
```sh
$ sed -i "" 's|REPLACE_IMAGE|quay.io/<user>/memcached-operator:v0.0.1|g' deploy/operator.yaml
```

The above command will replace the string `REPLACE_IMAGE` with the `<image>:<tag>` built above. Afterwards, verify that your `operator.yaml` file was updated successfully.

```yaml
serviceAccountName: memcached-operator
containers:
- name: memcached-operator
  # Replace this with the built image name
  image: quay.io/<user>/memcached-operator:v0.0.1
  command:
  - memcached-operator
  imagePullPolicy: Always
```

**IMPORTANT:** Ensure that your cluster is able to pull the image pushed to your registry.

The Deployment manifest is generated at `deploy/operator.yaml`. Be sure to update the deployment image as shown above since the default is just a placeholder.

Setup RBAC and deploy the memcached-operator:

```sh
$ kubectl create -f deploy/service_account.yaml
$ kubectl create -f deploy/role.yaml
$ kubectl create -f deploy/role_binding.yaml
$ kubectl create -f deploy/operator.yaml
```

**NOTE:** To apply the RBAC you need to be logged in `system:admin`. (E.g. By using for OCP: `oc login -u system:admin.`)

Verify that the `memcached-operator` Deployment is up and running:

```sh
$ kubectl get deployment
NAME                     DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
memcached-operator       1         1         1            1           1m
```

Verify that the `memcached-operator` pod is up and running:

```sh
$ kubectl get pod
NAME                                  READY     STATUS    RESTARTS   AGE
memcached-operator-7d76948766-nrcp7   1/1       Running   0          44s
```

**IMPORTANT:** Ensure that you built and pushed the image, and updated the `operator.yaml` file.   

Verify that the operator is running successfully by checking its logs.

```sh
$ kubectl logs memcached-operator-7d76948766-nrcp7
{"level":"info","ts":1580855834.104447,"logger":"cmd","msg":"Operator Version: 0.0.1"}
{"level":"info","ts":1580855834.1044931,"logger":"cmd","msg":"Go Version: go1.13.6"}
{"level":"info","ts":1580855834.104505,"logger":"cmd","msg":"Go OS/Arch: linux/amd64"}
{"level":"info","ts":1580855834.1045163,"logger":"cmd","msg":"Version of operator-sdk: v0.15.1"}
{"level":"info","ts":1580855834.1049826,"logger":"leader","msg":"Trying to become the leader."}
{"level":"info","ts":1580855834.4423697,"logger":"leader","msg":"No pre-existing lock was found."}
{"level":"info","ts":1580855834.447401,"logger":"leader","msg":"Became the leader."}
{"level":"info","ts":1580855834.7494223,"logger":"controller-runtime.metrics","msg":"metrics server is starting to listen","addr":"0.0.0.0:8383"}
{"level":"info","ts":1580855834.7497423,"logger":"cmd","msg":"Registering Components."}
{"level":"info","ts":1580855835.3955405,"logger":"metrics","msg":"Metrics Service object created","Service.Name":"memcached-operator-metrics","Service.Namespace":"default"}
{"level":"info","ts":1580855835.7000446,"logger":"cmd","msg":"Could not create ServiceMonitor object","error":"no ServiceMonitor registered with the API"}
{"level":"info","ts":1580855835.7005095,"logger":"cmd","msg":"Install prometheus-operator in your cluster to create ServiceMonitor objects","error":"no ServiceMonitor registered with the API"}
{"level":"info","ts":1580855835.7007008,"logger":"cmd","msg":"Starting the Cmd."}
{"level":"info","ts":1580855835.7014875,"logger":"controller-runtime.manager","msg":"starting metrics server","path":"/metrics"}
{"level":"info","ts":1580855835.702304,"logger":"controller-runtime.controller","msg":"Starting EventSource","controller":"memcached-controller","source":"kind source: /, Kind="}
{"level":"info","ts":1580855835.803201,"logger":"controller-runtime.controller","msg":"Starting EventSource","controller":"memcached-controller","source":"kind source: /, Kind="}
{"level":"info","ts":1580855835.9041016,"logger":"controller-runtime.controller","msg":"Starting Controller","controller":"memcached-controller"}
{"level":"info","ts":1580855835.9044445,"logger":"controller-runtime.controller","msg":"Starting workers","controller":"memcached-controller","worker count":1}
```

The following error will occur if your cluster was unable to pull the image:

```sh
$ kubectl get pod
NAME                                  READY     STATUS             RESTARTS   AGE
memcached-operator-6b5dc697fb-t62cv   0/1       ImagePullBackOff   0          2m
```

Following the logs in the error scenario described above.

```sh
$ kubectl logs memcached-operator-6b5dc697fb-t62cv
Error from server (BadRequest): container "memcached-operator" in pod "memcached-operator-6b5dc697fb-t62cv" is waiting to start: image can't be pulled
```

**NOTE:** Just for tests purposes make the image public and setting up the cluster to allow use insecure registry. ( E.g `--insecure-registry 172.30.0.0/16` )  

### 2. Run locally outside the cluster

This method is preferred during development cycle to deploy and test faster.

Run the operator locally with the default kubernetes config file present at `$HOME/.kube/config`:

```sh
$ operator-sdk run --local --watch-namespace=default
INFO[0000] Running the operator locally in namespace default.
{"level":"info","ts":1580761578.693055,"logger":"cmd","msg":"Operator Version: 0.0.1"}
{"level":"info","ts":1580761578.6931021,"logger":"cmd","msg":"Go Version: go1.13.1"}
{"level":"info","ts":1580761578.693109,"logger":"cmd","msg":"Go OS/Arch: darwin/amd64"}
{"level":"info","ts":1580761578.693113,"logger":"cmd","msg":"Version of operator-sdk: v0.15.1"}
...
```

You can use a specific kubeconfig via the flag `--kubeconfig=<path/to/kubeconfig>`.

## Create a Memcached CR

Create the example `Memcached` CR that was generated at `deploy/crds/cache.example.com_v1alpha1_memcached_cr.yaml`:

```sh
$ cat deploy/crds/cache.example.com_v1alpha1_memcached_cr.yaml
apiVersion: "cache.example.com/v1alpha1"
kind: "Memcached"
metadata:
  name: "example-memcached"
spec:
  size: 3

$ kubectl apply -f deploy/crds/cache.example.com_v1alpha1_memcached_cr.yaml
```

Ensure that the `memcached-operator` creates the deployment for the CR:

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
$ cat deploy/crds/cache.example.com_v1alpha1_memcached_cr.yaml
apiVersion: "cache.example.com/v1alpha1"
kind: "Memcached"
metadata:
  name: "example-memcached"
spec:
  size: 4

$ kubectl apply -f deploy/crds/cache.example.com_v1alpha1_memcached_cr.yaml
```

Confirm that the operator changes the deployment size:

```sh
$ kubectl get deployment
NAME                 DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
example-memcached    4         4         4            4           5m
```

### Cleanup

Delete the operator and its related resources:

```sh
$ kubectl delete -f deploy/crds/cache.example.com_v1alpha1_memcached_cr.yaml
$ kubectl delete -f deploy/operator.yaml
$ kubectl delete -f deploy/role_binding.yaml
$ kubectl delete -f deploy/role.yaml
$ kubectl delete -f deploy/service_account.yaml
$ kubectl delete -f deploy/crds/cache.example.com_memcacheds_crd.yaml
```

## Reference implementation

The above walkthrough follows a similar implementation process to the one used to produce the `memcached-operator` in the SDK [samples repo][repo_sdk_samples_memcached].

## Manage the operator using the Operator Lifecycle Manager

> NOTE: This section of the Getting Started Guide is out-of-date.
> We're working on some improvements to Operator SDK to streamline
> the experience of using OLM. For further information see, for example, this enhancement [proposal][sdk-integration-with-olm-doc]. 
> In the meantime, you might find the following documentation helpful: 

The previous section has covered manually running an operator. In the next sections, we will explore using the [Operator Lifecycle Manager][operator_lifecycle_manager] (OLM) which is what enables a more robust deployment model for operators being run in production environments.

OLM helps you to install, update, and generally manage the lifecycle of all of the operators (and their associated services) on a Kubernetes cluster. It runs as an Kubernetes extension and lets you use `kubectl` for all the lifecycle management functions without any additional tools.

**NOTE:** Various public, OLM-ready operator projects are available at [operatorhub.io][operator-hub-io]. 

### Generate an operator manifest

The first step to leveraging OLM is to create a [Cluster Service Version][csv_design_doc] (CSV) manifest. An operator manifest describes how to display, create and manage the application, in this case memcached, as a whole. It is required for OLM to function.

The Operator SDK CLI can generate CSV manifests via the following command:

```console
$ operator-sdk generate csv --csv-version 0.0.1 --update-crds
```

Several fields must be updated after generating the CSV. See the CSV generation doc for a list of [required fields][csv-fields], and the memcached-operator [CSV][memcached_csv] for an example of a complete CSV.

**NOTE:** You are able to preview and validate your CSV manifest syntax in the [operatorhub.io CSV Preview][operator-hub-io-preview] tool.

### Testing locally

The next step is to ensure your project deploys correctly with OLM and runs as expected. Follow this [testing guide][testing-operators] to deploy and test your operator.

**NOTE:** Also, check out some of the new OLM integrations in operator-sdk:
- [`operator-sdk olm`][sdk-olm-cli] to install and manage an OLM installation in your cluster.
- [`operator-sdk run --olm`][sdk-run-cli] to run your operator using the CSV generated by `operator-sdk generate csv`.
- [`operator-sdk bundle`][sdk-bundle-cli] to create and validate operator bundle images.

### Promoting operator standards

We recommend running `operator-sdk scorecard` against your operator to see whether your operator's OLM integration follows best practices. For further information on running the scorecard and results, see the [scorecard documentation][scorecard-doc].

**NOTE:** the scorecard is undergoing changes to give informative and helpful feedback. The original scorecard functionality will still be available while and after changes are made.

## Conclusion

Hopefully, this guide was an effective demonstration of the value of the Operator Framework for building and managing operators. There is much more that we left out in the interest of brevity. The Operator Framework and its components are open source, so please feel encouraged to jump into each individually and learn what else you can do. If you want to discuss your experience, have questions, or want to get involved, join the Operator Framework [mailing list][mailing_list].

<!---  Reference URLs begin here -->

[operator_group_doc]: https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/design/operatorgroups.md
[csv_design_doc]: https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/design/building-your-csv.md
[csv_generation_doc]: https://github.com/operator-framework/operator-sdk/blob/master/doc/user/olm-catalog/generating-a-csv.md
[org_operator_framework]: https://github.com/operator-framework/
[site_blog_post]: https://coreos.com/blog/introducing-operator-framework
[operator_sdk]: https://github.com/operator-framework/operator-sdk
[operator_lifecycle_manager]: https://github.com/operator-framework/operator-lifecycle-manager
[site_memcached]: https://memcached.org/
[doc_sdk_user_guide]: https://github.com/operator-framework/operator-sdk/blob/master/doc/user-guide.md
[doc_sdk_install_instr]: https://github.com/operator-framework/operator-sdk/blob/master/doc/user-guide.md#install-the-operator-sdk-cli
[doc_olm_install_instr]: https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/install/install.md
[layout_doc]: https://github.com/operator-framework/operator-sdk/blob/master/doc/project_layout.md
[manager_go_doc]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/manager#Manager
[controller_go_doc]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg#hdr-Controller
[memcached_controller]: https://github.com/operator-framework/operator-sdk/blob/master/example/memcached-operator/memcached_controller.go.tmpl
[request_go_doc]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/reconcile#Request
[result_go_doc]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/reconcile#Result
[doc_client_api]: https://github.com/operator-framework/operator-sdk/blob/master/doc/user/client.md
[repo_sdk_samples_memcached]: https://github.com/operator-framework/operator-sdk-samples/tree/master/go/memcached-operator/
[mailing_list]: https://groups.google.com/forum/#!forum/operator-framework
[memcached_csv]: https://github.com/operator-framework/operator-sdk/blob/master/test/test-framework/deploy/olm-catalog/memcached-operator/0.0.3/memcached-operator.v0.0.3.clusterserviceversion.yaml
[testing-operators]: https://github.com/operator-framework/community-operators/blob/master/docs/testing-operators.md
[sdk-integration-with-olm-doc]: https://github.com/operator-framework/operator-sdk/blob/master/doc/proposals/sdk-integration-with-olm.md
[sdk-olm-cli]: https://github.com/operator-framework/operator-sdk/blob/master/doc/cli/operator-sdk_olm.md
[sdk-run-cli]: https://github.com/operator-framework/operator-sdk/blob/master/doc/cli/operator-sdk_run.md
[sdk-bundle-cli]: https://github.com/operator-framework/operator-sdk/blob/master/doc/cli/operator-sdk_bundle.md
[operator-hub-io]: https://operatorhub.io/ 
[operator-hub-io-preview]: https://operatorhub.io/preview
[scorecard-doc]: https://github.com/operator-framework/operator-sdk/blob/master/doc/test-framework/scorecard.md
[csv-fields]: https://github.com/operator-framework/operator-sdk/blob/master/doc/user/olm-catalog/generating-a-csv.md#csv-fields