---
title: CRD Scope
linkTitle: CRD Scope
weight: 60
---

## Overview

The CustomResourceDefinition (CRD) scope can also be changed for cluster-scoped operators so that there is only a single 
instance (for a given name) of the CRD to manage across the cluster.

The CRD manifests are generated in `config/crd/bases`. For each CRD that needs to be cluster-scoped, its manifest 
should specify `spec.scope: Cluster`.

To ensure that the CRD is always generated with `scope: Cluster`, add the marker 
`// +kubebuilder:resource:path=<resource>,scope=Cluster`, or if already present replace `scope={Namespaced -> Cluster}`, 
above the CRD's Go type definition in `api/<version>/<kind>_types.go` or `apis/<group>/<version>/<kind>_types.go` 
if you are using the `multigroup` layout. Note that the `<resource>` 
element must be the same lower-case plural value of the CRD's Kind, `spec.names.plural`. 

## CRD cluster-scoped usage 

A cluster scope is ideal for operators that manage custom resources (CR's) that can be created in more than 
one namespace in a cluster. 

**NOTE**: When a `Manager` instance is created in the `main.go` file, it receives the namespace(s) as Options. 
These namespace(s) should be watched and cached for the Client which is provided by the Controllers. Only clients 
provided by cluster-scoped projects where the `Namespace` attribute is `""` will be able to manage cluster-scoped CRD's. 
For more information see the [Manager][manager_user_guide] topic in the user guide and the 
[Manager Options][manager_options].

## Example for changing the CRD scope from Namespaced to Cluster 

- Check the `spec.names.plural` in the  CRD's Kind YAML file

* `/config/crd/bases/cache.example.com_memcacheds.yaml`
```YAML
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  annotations:
    controller-gen.kubebuilder.io/version: v0.2.5
  creationTimestamp: null
  name: memcacheds.cache.example.com
spec:
  group: cache.example.com
  names:
    kind: Memcached
    listKind: MemcachedList
    plural: memcacheds
    singular: memcached
  scope: Namespaced
  subresources:
    status: {}
...   
``` 

- Update the `apis/<version>/<kind>_types.go` by adding the 
marker `// +kubebuilder:resource:path=<resource>,scope=Cluster`

* `api/v1alpha1/memcached_types.go`

```Go
// Memcached is the Schema for the memcacheds API
// +kubebuilder:resource:path=memcacheds,scope=Cluster
type Memcached struct {
  metav1.TypeMeta   `json:",inline"`
  metav1.ObjectMeta `json:"metadata,omitempty"`

  Spec   MemcachedSpec   `json:"spec,omitempty"`
  Status MemcachedStatus `json:"status,omitempty"`
}
``` 
- Run `make manifests`, to update the CRD manifest with the cluster scope setting, as in the following example:
  
* `/config/crd/bases/cache.example.com_memcacheds.yaml`

```YAML
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  annotations:
    controller-gen.kubebuilder.io/version: v0.2.5
  creationTimestamp: null
  name: memcacheds.cache.example.com
spec:
  group: cache.example.com
  names:
    kind: Memcached
    listKind: MemcachedList
    plural: memcacheds
    singular: memcached
  scope: Cluster
  subresources:
    status: {}
...   
``` 
  
[RBAC]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/
[manager_user_guide]:/docs/building-operators/golang/tutorial/#manager
[manager_options]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/manager#Options
