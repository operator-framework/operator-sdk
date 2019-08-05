## Operator scope

A namespace-scoped operator watches and manages resources in a single namespace, whereas a cluster-scoped operator watches and manages resources cluster-wide. Namespace-scoped operators are preferred because of their flexibility. They enable decoupled upgrades, namespace isolation for failures and monitoring, and differing API definitions.

However, there are use cases where a cluster-scoped operator may make sense. For example, the [cert-manager](https://github.com/jetstack/cert-manager) operator is often deployed with cluster-scoped permissions and watches so that it can manage issuing certificates for an entire cluster.

The SDK scaffolds operators to be namespaced by default but with a few modifications to the default manifests the operator can be run as cluster-scoped.

* `deploy/operator.yaml`:
  * Set `WATCH_NAMESPACE=""` to watch all namespaces instead of setting it to the pod's namespace
* `deploy/role.yaml`:
  * Use `ClusterRole` instead of `Role`
* `deploy/role_binding.yaml`:
  * Use `ClusterRoleBinding` instead of `RoleBinding`
  * Use `ClusterRole` instead of `Role` for `roleRef`
  * Set the subject namespace to the namespace in which the operator is deployed.

### CRD scope

Additionally the CustomResourceDefinition (CRD) scope can also be changed for cluster-scoped operators so that there is only a single instance (for a given name) of the CRD to manage across the cluster.

> **NOTE**: Cluster-scoped CRDs are **NOT** supported with the Helm operator. While Helm releases can create cluster-scoped resources, Helm's design requires the release itself to be created in a specific namespace. Since the Helm operator uses a 1-to-1 mapping between a CR and a Helm release, Helm's namespace-scoped release requirement extends to Helm operator's namespace-scoped CR requirement.

For each CRD that needs to be cluster-scoped, update its manifest to be cluster-scoped.

* `deploy/crds/<group>_<version>_<kind>_crd.yaml`
  * Set `spec.scope: Cluster`

To ensure that the CRD is always generated with `scope: Cluster`, add the tag `// +genclient:nonNamespaced` above the CRD's Go type defintion in `pkg/apis/<group>/<version>/<kind>_types.go`.


### Example for cluster scoped operator

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
* `deploy/crds/cache_v1alpha1_memcached_crd.yaml`
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
* `pkg/apis/cache/v1alpha1/memcached_types.go`
    ```Go
    // +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

    // Memcached is the Schema for the memcacheds API
    // +k8s:openapi-gen=true
    // +genclient:nonNamespaced
    type Memcached struct {
      metav1.TypeMeta   `json:",inline"`
      metav1.ObjectMeta `json:"metadata,omitempty"`

      Spec   MemcachedSpec   `json:"spec,omitempty"`
      Status MemcachedStatus `json:"status,omitempty"`
    }
    ```
