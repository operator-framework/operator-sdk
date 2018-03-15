Operator-SDK Query APIs Design Doc
----
* Author(s): Fanmin Shi
* Approver: Haseeb
* Status: Draft
* Implemented in: Golang
* Last updated: 03/14/2018

## Abstract

In order to build a functional operator, the ability to reason the state of world is a must. For instance, 
for a given etcd cluster, etcd-operator needs to know the number of live etcd pods so that it can reconcile the etcd cluster 
based on that. In addition, many operators need to retrieve the kubernetes object such as secret, status, and more to 
manage their applications. Hence, the operator-sdk needs to provide some sort of Query APIs that allow a operator to
execute their next step base on state and provided resource of the world.

## Background

The provided Kubernetes client-go does provide a set of APIs that allows user to reason about the state of the world via accessing the kubernetes objects.
However, those APIs are too lower level, lack of documentation, and verbose to use. For example, If I want to get a specific `Deployment` Object,
I need to do the following:
* Create a kubernete client `KubeCli` that implements `kubernetes.Interface`.
* Select the right API Version `KubeCli.Apps()...` that `Deployment` type resides in.
* Enter the namespace to retrieve the Deployment Object: `KubeCli.Apps().Deployments("default")...`.
* Enter the "name" of the object: `KubeCli.Apps().Deployments("default").Get("name",...)`.
* Last, figures of what `v1.GetOptions{}` to fill: `KubeCli.Apps().Deployments("default").Get("name", v1.GetOptions{})`

The above code steps seem too verbose to retrieve a `Deployment` object. Can we do better?

In addition, if I want get a Custom Resource such as `EtcdCluster`, I can't reuse the `KubeCli` defined in previous steps.
I need to create a new custom resource client that implements `versioned.Interface`. And before that, I need to run Kubernete code-generation on that Custom Resource to generate a specific client for `EtcdCluster` for creating the custom resource client.

## Proposal

To simplify the retrieval of Kubernete object and objects, I propose a Query APIs consisting of `Get()` and `List()` where `Get()` gets a Kubernete Object and `List()` lists a list of Kubernete object of the same type.

One interesting observation on a Go kubernete Object such as `Deployment`. The type of object is mostly tied to a single specific
`api-verison`. For example, `Deployment` type imported from `apps_v1 "k8s.io/api/apps/v1"` has `api-version` of `apps/v1`. Using this fact, we can infer the `api-version` from the given Kubernete Object and can use that to construct a
specific resource client which allows us to get the corresponding Kubernete object from the api server. Hence, I propose the following generic `Get()` that takes in a kubernete object along with few minimum required arguments where the `Get()` infers required infos from the given kubernete object to construct a resource client, uses the resource client to get kubernetes object data from the api server, and finally unmarshals the data into the pass-in kubernete object.

```go
// Get gets the kubernetes object of the type same as "into" object given the "name" and "namespace"
// and then unmarshals the retrieved data into the "into" object.
//
// Note: Get infers Group, Kind, and Version (e.g <apps, Deployment, v1>)
// from type of "into" object (e.g Deployment{}) and uses those to construct
// the corresponding client which is then used to retrieve
// the correct kubernetes object.
// However, the "into" object type can many <Group, Kind, and Version> 
// (e.g [<example.com, app, v1>, <example.com, app, v2>] ) if the object type
// is registered with multiple versions. If that's the case, Get returns an error.
func Get(ctx sdkTypes.Context, name string, namespace string, into runtime.Object, opts ...GetOption) error
```

The `Get` accepts `GetOption` as part of argument. In this way, we follow the open-close principle 
which allows us to extend the `Get` api with unforseen features without modify the api itself:

```go
// GetOp represents an Operation that Get can execute.
type GetOp struct {
...
}

// GetOption configures Get.
type GetOption func(*GetOp)

// WithResourceVersion specifies ResourceVersion for GetOptions.
func WithResourceVersion(rv string) GetOption {
	return func(op *GetOp) {...}
}
// More options
...
```

Example usage:
```go
d := &apps_v1.Deployment{}
err := sdk.Get(ctx, "example", "default", d, op.WithResourceVersion("0"))
if err != nil {
   // handle error
}
fmt.Printf("Deployment %+v", d)
// do something with "d".
```

Base on the same principle of `Get()`, the `List()` is defined as following:

```go
// List List the kubernetes objects of the type same as "into" object given the "namespace"
// and then unmarshals the retrived blob into the "into" object.
//
// Note: List infers Group, Kind, and Version (e.g <apps, Deployment, v1>)
// from type of "into" object (e.g Deployment{}) and uses those to construct
// the corresponding client which is then used to retrieve
// the correct kubernetes object.
// However, the "into" object type can many <Group, Kind, and Version> 
// (e.g [<example.com, app, v1>, <example.com, app, v2>] ) if the object type
// is registered with multiple versions. If that's the case, List returns an error.
func List(ctx sdkTypes.Context, namespace string, obj runtime.Object, opts ...ListOption) error
```

The `List` accepts `ListOption` as part of argument. In this way, we follow the open-close principle 
which allows us to extend the `ListOption` api with unforseen features without modify the api itself:

```go
// ListOp represents an Operation that List can execute.
type ListOp struct {
...
}

// ListOption configures List.
type ListOption func(*ListOp)

// WithLabel specifies a label for ListOptions.
func WithLabel(key, val string) ListOption {
	return func(op *ListOp) {...}
}
// more ListOptions
...
```

Example usage:

```go
dl := &apps_v1.DeploymentList{}
err = sdk.List(ctx, o.Namespace, dl, op.WithLabel("name", "app-operator"))
if err != nil {
    // handle error
}
fmt.Printf("Deployment List %+v", dl)
// do something with Deployment List.
```

**Caveat** Most of Go Kubernete Object types have only one api-version. However, 
a type can have many api-versions. That causes Get() and List() to be confused
on which api-version to use when constructing a resource client.

For example, suppose I have an AppService Custom Resource.

```go
// type.go
type AppService struct {
...
}
```

I can register the `AppService` type twice with different Versions:

```go
// register.go
var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	SchemeGroupVersion   = schema.GroupVersion{Group: "example.com", Version: "v1"}
	SchemeGroupVersionV2 = schema.GroupVersion{Group: "example.com", Version: "v2"}
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&Memcached{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	scheme.AddKnownTypes(SchemeGroupVersionV2,
		&Memcached{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersionV2)
	return nil
}
```

When looking up api-version using `scheme.ObjectKinds` for the type `AppService`, it has
two api-versions `cache.example.com/v1alpha1` and `cache.example.com/v1alpha2`:

```go
// test.go
func TestKind(t *testing.T) {
	scheme := runtime.NewScheme()
	AddToScheme(scheme)
	ks, _, _ := scheme.ObjectKinds(&AppService{})
    fmt.Printf("%v \n", ks)
    // Output:
    // [example.com/v1, Kind=AppService example.com/v1, Kind=AppService]
}
```

The above scenario can cause `Get()` and `List()` to be confused on which api-version to use  on constructing a resource client. However,
this does not happen in practice because user can't create CRD for a given Kind (e.g AppService) with more than one api-version (e.g example.com/v1 and example.com/v2). Also I don't think any of the core Kubernete types has more than one api-version. For example, the `Deployment` type imported from `"k8s.io/api/apps/v1"` has an api-verion `apps/v1` and not more.

## Rationale

Alternative approaches:

I have thought about different interface for `Get()` and `List()`:

Option 1:

Use Kubernete specific GetOptions for `Get()` and `ListOptions` for `List()`:
```
Get(ctx sdkTypes.Context, name string, namespace string, into runtime.Object, opts meta_v1.GetOptions) error
List(ctx sdkTypes.Context, namespace string, into runtime.Object, opts meta_v1.ListOptions{}) error
```

Pro:
* No need to define option wrapper at interface level
* Simpler to implement.

Con:
* Force user to create an Option even if user might want too; hence more code to write.
* User needs to read Kubernete specific `GetOptions` and `ListOptions` doc instead of reasoning only the provided Option Wrapper.
* Doesn't allow any modification to API in the future.

Comment:

Combining a higher level API with low level API is not elegant. The purpose of higher level API is to abstract away
the lower one so that the user can have a more intuitive and descriptive understanding of the API.
For example, etcd clientV3 wraps atop of the generated protobuf GRPC client. many of clientV3 APIs use options providing a better documentation to construct the lower level protobuf request message.

Option 2:

Add `api-version` and `kind` arguments to `Get()` and `List()` so that
those functions doesn't have to infer those base on the Go kubernete type; 
Hence, no conflicting api-versions can't ever happen as described in the **Caveat** section above. 

```
Get(ctx sdkTypes.Context, name string, namespace string, kind string, apiVersion string, into runtime.Object, opts meta_v1.GetOptions) error
List(ctx sdkTypes.Context, namespace string, kind string, apiVersion string, into runtime.Object, opts meta_v1.ListOptions{}) error
```

Pro:
* No api-versions conflict possible

Con:
* Both `Get()` and `List()` have too many arguments which don't really make Query APIs any simpler to reason than that of kubernete client-go.

Comment:
As discussed in the **Caveat** section, the multiple api-versions issue for a given kubernete object type doesn't really happen
in pratice. Hence, we don't have to worry about it much. If it becomes a problem, we can alway add additional `GetOption` and `ListOption` such as `WithAPIVersion()` and `WithKind()` to allow user to set those specifically. 

## Implementation

Actions Items:
* Implement the `GetOption`
* Implement and Test `Get`
* Implement the `ListOption`
* Implement and Test `List`
