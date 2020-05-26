---
title: CRD scope with Operator SDK
linkTitle: CRD Scope
weight: 60
---

## Overview

The CustomResourceDefinition (CRD) scope can also be changed for cluster-scoped operators so that there is only a single 
instance (for a given name) of the CRD to manage across the cluster.

**NOTE**: Cluster-scoped CRDs are **NOT** supported with the Helm operator. While Helm releases can create 
cluster-scoped resources, Helm's design requires the release itself to be created in a specific namespace. Since the 
Helm operator uses a 1-to-1 mapping between a CR and a Helm release, Helm's namespace-scoped release requirement 
extends to Helm operator's namespace-scoped CR requirement.

For each CRD that needs to be cluster-scoped, update its manifest to be cluster-scoped.

* `deploy/crds/<full group>_<resource>_crd.yaml`
  * Set `spec.scope: Cluster`

To ensure that the CRD is always generated with `scope: Cluster`, add the marker 
`// +kubebuilder:resource:path=<resource>,scope=Cluster`, or if already present replace `scope={Namespaced -> Cluster}`, 
above the CRD's Go type definition in `pkg/apis/<group>/<version>/<kind>_types.go`. Note that the `<resource>` 
element must be the same lower-case plural value of the CRD's Kind, `spec.names.plural`. 

## CRD cluster-scoped usage 

A cluster scope is ideal for operators that manage custom resources (CR's) that can be created in more than one namespace in a cluster. 

**NOTE**: When a `Manager` instance is created in the `main.go` file, it receives the namespace(s) as Options. 
These namespace(s) should be watched and cached for the Client which is provided by the Controllers. Only clients 
provided by cluster-scoped projects where the `Namespace` attribute is `""` will be able to manage cluster-scoped CRD's. 
For more information see the [Manager][manager_user_guide] topic in the user guide and the 
[Manager Options][manager_options].

## Example for changing the CRD scope from Namespaced to Cluster 

The following example is for Go based-operators. `scope: Cluster` must set manually for Helm and Ansible based-operators.

- Check the `spec.names.plural` in the  CRD's Kind YAML file

* `deploy/crds/cache_v1alpha1_memcached_crd.yaml`
    ```YAML
    apiVersion: apiextensions.k8s.io/v1beta1
    kind: CustomResourceDefinition
    metadata:
      name: memcacheds.cache.example.com
    spec:
      group: cache.example.com
      names:
        kind: Memcached
        listKind: MemcachedList
        plural: memcacheds
        singular: memcached
      scope: Namespaced
    ``` 

- Update the `pkg/apis/<group>/<version>/<kind>_types.go` by adding the 
marker `// +kubebuilder:resource:path=<resource>,scope=Cluster`

* `pkg/apis/cache/v1alpha1/memcached_types.go`
    ```Go
    // +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

    // Memcached is the Schema for the memcacheds API
    // +kubebuilder:resource:path=memcacheds,scope=Cluster
    type Memcached struct {
      metav1.TypeMeta   `json:",inline"`
      metav1.ObjectMeta `json:"metadata,omitempty"`

      Spec   MemcachedSpec   `json:"spec,omitempty"`
      Status MemcachedStatus `json:"status,omitempty"`
    }
    ``` 
- Execute the command `operator-sdk generate crds`, then you should be able to check that the CRD was updated with the cluster scope as in the following example:
  
* `deploy/crds/cache.example.com_memcacheds_crd.yaml`
    ```YAML
    apiVersion: apiextensions.k8s.io/v1beta1
    kind: CustomResourceDefinition
    metadata:
      name: memcacheds.cache.example.com
    spec:
      group: cache.example.com
      ...
      scope: Cluster
    ```
  
[RBAC]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/
[manager_user_guide]: /docs/golang/quickstart/#manager
[manager_options]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/manager#Options
