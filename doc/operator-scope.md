
# Operators and CRD scope with Operator SDK

- [Namespace-scoped operator usage](#namespace-scoped-operator-usage)
- [Cluster-scoped operator usage](#cluster-scoped-operator-usage)
  - [Changed required for a cluster-scoped operator](#changed-required-for-a-cluster-scoped-operator)
  - [Example for cluster-scoped operator](#example-for-cluster-scoped-operator)
- [CRD scope](#crd-scope)
  - [CRD cluster-scoped usage](#crd-cluster-scoped-usage)
  - [Example for changing the CRD scope from namespace to cluster](#example-for-changing-the-crd-scope-from-namespace-to-cluster)

## Overview

A namespace-scoped operator watches and manages resources in a single namespace, whereas a cluster-scoped operator watches and manages resources cluster-wide. Namespace-scoped operators are preferred because of their flexibility. They enable decoupled upgrades, namespace isolation for failures and monitoring, and differing API definitions.

However, there are use cases where a cluster-scoped operator may make sense. For example, the [cert-manager](https://github.com/jetstack/cert-manager) operator is often deployed with cluster-scoped permissions and watches so that it can manage issuing certificates for an entire cluster.

## Namespace-scoped operator usage

This scope is ideal for operator projects which will control resources just in one namespace, which is where the operator is deployed.

> **NOTE:** Initial projects created by `operator-sdk` are namespace-scoped by default which means that it will NOT have a `ClusterRole` defined in the `deploy/role_binding.yaml`.

## Cluster-scoped operator usage

This scope is ideal for operator projects which will control resources in more than one namespace.

### Changes required for a cluster-scoped operator

The SDK scaffolds operators to be namespaced by default but with a few modifications to the default manifests the operator can be run as cluster-scoped.

* `deploy/operator.yaml`:
  * Set `WATCH_NAMESPACE=""` to watch all namespaces instead of setting it to the pod's namespace
* `deploy/role.yaml`:
  * Use `ClusterRole` instead of `Role`
* `deploy/role_binding.yaml`:
  * Use `ClusterRoleBinding` instead of `RoleBinding`
  * Use `ClusterRole` instead of `Role` for `roleRef`
  * Set the subject namespace to the namespace in which the operator is deployed.

### Example for cluster-scoped operator

With the above changes the specified manifests should look as follows:

* `deploy/operator.yaml`:
    ```YAML
    apiVersion: apps/v1
    kind: Deployment
    ...
    spec:
      ...
      template:
        ...
        spec:
          ...
          serviceAccountName: memcached-operator
          containers:
          - name: memcached-operator
            ...
            env:
            - name: WATCH_NAMESPACE
              value: ""
    ```
* `deploy/role.yaml`:
    ```YAML
    apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRole
    metadata:
      name: memcached-operator
    ...
    ```
* `deploy/role_binding.yaml`:
    ```YAML
    kind: ClusterRoleBinding
    apiVersion: rbac.authorization.k8s.io/v1
    metadata:
      name: memcached-operator
    subjects:
    - kind: ServiceAccount
      name: memcached-operator
      namespace: <operator-namespace>
    roleRef:
      kind: ClusterRole
      name: memcached-operator
      apiGroup: rbac.authorization.k8s.io
    ```

## CRD scope

Additionally the CustomResourceDefinition (CRD) scope can also be changed for cluster-scoped operators so that there is only a single instance (for a given name) of the CRD to manage across the cluster.

> **NOTE**: Cluster-scoped CRDs are **NOT** supported with the Helm operator. While Helm releases can create cluster-scoped resources, Helm's design requires the release itself to be created in a specific namespace. Since the Helm operator uses a 1-to-1 mapping between a CR and a Helm release, Helm's namespace-scoped release requirement extends to Helm operator's namespace-scoped CR requirement.

For each CRD that needs to be cluster-scoped, update its manifest to be cluster-scoped.

* `deploy/crds/<full group>_<resource>_crd.yaml`
  * Set `spec.scope: Cluster`

To ensure that the CRD is always generated with `scope: Cluster`, add the tag `// +kubebuilder:resource:path=<resource>,scope=Cluster`, or if already present replace `scope={Namespaced -> Cluster}`, above the CRD's Go type definition in `pkg/apis/<group>/<version>/<kind>_types.go`. Note that the `<resource>` element must be the same lower-case plural value of the CRD's Kind, `spec.names.plural`. 

### CRD cluster-scoped usage 

This scope is ideal for the cases where an instance(CR) of some Kind(CRD) will be used in more than one namespace instead of a specific one. 

> **NOTE**: When a `Manager` instance is created in the `main.go` file, it receives the namespace(s) as Options. These namespace(s) should be watched and cached for the Client which is provided by the Controllers. Only clients provided by cluster-scoped projects where the `Namespace` attribute is `""` will be able to manage cluster-scoped CRD's. For more information see the [Manager][manager_user_guide] topic in the user guide and the [Manager Options][manager_options].

### Example for changing the CRD scope from namespace to cluster 

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

- Update the `pkg/apis/<group>/<version>/<kind>_types.go` by adding the tag `// +kubebuilder:resource:path=<resource>,scope=Cluster`

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
[manager_user_guide]: ./user-guide.md#manager
[manager_options]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/manager#Options
