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

For each CRD that needs to be cluster-scoped, update its manifest to be cluster-scoped.

* `deploy/crds/<group>_<version>_<kind>_crd.yaml`
  * Set `spec.scope: Cluster`
