---
title: Composable integration in SDK
authors:
  - "dettori@us.ibm.com"	
  - "luan@us.ibm.com"
  - "roytman@il.ibm.com"
  - "tardieu@us.ibm.com"
  - "mvaziri@us.ibm.com"
  
 
reviewers:
  - TBD
approvers:
  - TBD
creation-date: 2019-12-16
last-updated: 2019-12-16
status: implemented
see-also:
replaces:
superseded-by:
---

# Composable integration in SDK

## Release Signoff Checklist

- \[x\] Enhancement is `implementable`
- \[x\] Design details are appropriately documented from clear requirements
- \[ \] Test plan is defined
- \[ \] Graduation criteria for dev preview, tech preview, GA
- \[ \] User-facing documentation is created in [operator-sdk/doc][operator-sdk-doc]


## Summary

Kubernetes object specifications often require constant values for their fields. When deploying an entire application
with many different resources, this limitation often results in the need for staged deployments, because some resources have to be deployed first in order to determine what data to provide for the specifications of dependent resources. This undermines the declarative nature of Kubernetes object specification and requires workflows, manual step-by-step instructions and/or brittle automated scripts for the deployment of applications as a whole.

The Composable SDK can be used to add cross-resource references to any existing CRD, so that values no longer
need to be hardwired. This feature allows dynamic configuration of a resource, meaning that its fields can be 
resolved after it has been deployed. 

See this [tutorial](https://github.com/IBM/composable/blob/master/sdk/docs/tutorial.md), in which we add cross-references to the `memcached-operator` using the Composable SDK.

The Composable SDK has been implemented and open-sourced at https://github.com/IBM/composable/tree/master/sdk.
It is also related to the Composable Operator released on operatorhub.io:
https://github.com/IBM/composable.


## Motivation

To add support for cross-resource references in CRD definitions and controllers (Composable SDK) and make this
functionality available through the Operator-SDK.

## Goals

- Provide cross-resource reference types, which can be used in CRD type definitions.
- Provide resolution functions, which can be called from within a reconciler to resolve cross-resource references.


## Proposal

### User Story 1

As an operator developer, I wish not to have the value of a field be hard-wired. These are typically
values that are computed dynamically when an application is deployed. For example, the admin URL for a Kafka
deployment is known only after Kafka has been successfully deployed. If my CRD requires the URL, then
rather that requiring a string value for that field, I can define it to be a reference to another object
(perhaps the data is in the status field of the Kubernetes object managing Kafka).

### User Story 2

Often the yamls for a collection of resources require the same data in many places (e.g., a host name or port number).
Rather than hardwiring the same value everywhere, I can define cross-resource references to a unique configmap that contains 
the data. This makes it easier to change the data if needed, and can allow all resources to be configured
dynamically (i.e. changes in the configmap are absorbed and resource that point to it are updated).


### Implementation Details/Notes/Constraints (optional)

The Composable SDK offers the following types to be used in a CRD definition:

```golang
type ObjectRef struct {
	GetValueFrom ComposableGetValueFrom `json:"getValueFrom"`
}

type ComposableGetValueFrom struct {
	Kind               string   `json:"kind"`
	APIVersion         string   `json:"apiVersion,omitempty"`
	Name               string   `json:"name,omitempty"`
	Labels             []string `json:"labels,omitempty"`
	Namespace          string   `json:"namespace,omitempty"`
	Path               string   `json:"path"`
	FormatTransformers []string `json:"format-transformers,omitempty"`
}
```

An `ObjectRef` can be used to specify the type of any field of a CRD definition, allowing the value to be determined dynamically.
For a detailed explanation of how to specify an object reference according to this schema, see [here](https://github.com/IBM/composable/blob/master/README.md#getvaluefrom-elements).

The Composable SDK offers the following types to be used as part of a Reconciler in a controller:

```golang
type ResolveObject interface {
	ResolveObject(ctx context.Context, in, out interface{}) error
}

type KubernetesResourceResolver struct {
	Client          client.Client
	ResourcesClient discovery.ServerResourcesInterface
}
```

The interface `ResolveObject` provides a function to resolve object references (see below). The struct `KubernetesResourceResolver`
implements it and can be used as part of a Reconciler struct in a CRD controller (see [tutorial](./docs/tutorial.md)). It requires a `Client` and a
`ServerResourceInterface` used to query Kubernetes about existing resources.

A `ServerResourceInterface` can be instantiated as follows:

```golang
discovery.NewDiscoveryClientForConfigOrDie(cfg)
```

where `discovery` is the package `k8s.io/client-go/discovery`, and `cfg` is a `rest.Config`.


The Composable SDK offers the following function for resolving the value of cross-resource references.

```golang
func (k KubernetesResourceResolver) ResolveObject(ctx context.Context, object, resolved interface{}) error {
```

The function `ResolveObject` takes a context, an `object` to resolve, and a blank object
`resolved` that will contain the result of resolving cross-resource references. 
It assumes that the input object has a namespace, which is then used as the default namespace when references 
do not specify one. This function will cast the result to the type of the `resolved` object, provided that
appropriate data transforms have been included in the reference definitions (see [tutorial](./docs/tutorial.md) for an example).

The `ResolveObject` function is one more way that the user can access Kubernetes objects, similar to APIs
already available in Operator-SDK, such as `r.Get(...)`, `r.Update(...)`, etc...

This function uses caching for looking up objects in order
to ensure that a consistent view of the data is obtained. If any data is not available at the time of the lookup,
it returns an error. So this function either resolves the entire object or it doesn't -- there are no partial results.

The return value of `ResolveObject` is an `error` and the Composable SDK offers a series of functions to determine
the nature of the error. This is used to decide whether the error needs to be returned by the Reconcile function or not.

```golang
func IsIllFormedRef(err error) bool 

func IsKindNotFound(err error) bool 

func IsObjectNotFound(err error) bool 

func IsValueNotFound(err error) bool 

func IsRefNotFound(err error) bool 
```

Function `IsIllFormedRef` indicates that that a cross-resource reference is ill-formed (in which case retrying reconciliation
would probably not help). Function `IsKindNotFound` indicates that the kind of the reference does not exist.
`IsObjectNotFound` indicates that the object itself does not exist, and `IsValueNotFound` that the value within the object
does not exist. Finally, `IsRefNotFound` is true if either `IsKindNotFound`, `IsObjectNotFound`, or `IsValueNotFound` are true.


### Risks and Mitigations

Using the Composable SDK means that the CRD needs to have permission to access objects that are being referenced.
This means that the operator developer needs to consider what kinds of objects will be permitted for references and
add appropriate permissions for the RBAC rules of the CRD.

## Design Details

### Test Plan

**Note:** *Section not required until targeted at a release.*


### Graduation Criteria

**Note:** *Section not required until targeted at a release.*


## Implementation History

Composable SDK is currently available (version sdk/v0.1.3) at: https://github.com/IBM/composable/tree/master/sdk.


## Drawbacks


## Alternatives

The operator developer can develop their own types and resolution function, but they would have to duplicate
the capability of caching and all-or-nothing approach. These functionalities are best provided as a library.

The operator developer may also use the Composable Operator to wrap an existing resource (without any changes to the CRD) and allow it to have cross-resource references (https://github.com/IBM/composable). 
The Composable SDK, as opposed to the Composable Operator,  allows a tighter integration
of references without the need for wrapping resources.



[operator-sdk-doc]:  ../../doc
