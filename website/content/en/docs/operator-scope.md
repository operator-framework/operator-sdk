---
title: Operators and CRD scope with Operator SDK
linkTitle: Operator Scope
weight: 50
---

## Overview

A namespace-scoped operator watches and manages resources in a single namespace, whereas a cluster-scoped operator watches and manages resources cluster-wide. Namespace-scoped operators are preferred because of their flexibility. They enable decoupled upgrades, namespace isolation for failures and monitoring, and differing API definitions.

However, there are use cases where a cluster-scoped operator may make sense. For example, the [cert-manager](https://github.com/jetstack/cert-manager) operator is often deployed with cluster-scoped permissions and watches so that it can manage issuing certificates for an entire cluster.

**NOTE**: CustomResourceDefinition (CRD) scope can also be changed to cluster-scoped. See the [CRD scope][crd-scope-doc] document for more details.

## Namespace-scoped operator usage

This scope is ideal for operator projects which will control resources just in one namespace, which is where the operator is deployed.

**NOTE:**  Projects created by `operator-sdk` are namespace-scoped by default which means that they will NOT have a `ClusterRole` defined in `deploy/`.

## Cluster-scoped operator usage

This scope is ideal for operator projects which will control resources in more than one namespace.

### Changes required for a cluster-scoped operator

The SDK scaffolds operators to be namespaced by default but with a few modifications to the default manifests the operator can be run as cluster-scoped.

* `deploy/operator.yaml`:
  * Set `WATCH_NAMESPACE=""` to watch all namespaces instead of setting it to the pod's namespace
  * Set `metadata.namespace` to define the namespace where the operator will be deployed.
* `deploy/role.yaml`:
  * Use `ClusterRole` instead of `Role`
* `deploy/role_binding.yaml`:
  * Use `ClusterRoleBinding` instead of `RoleBinding`
  * Use `ClusterRole` instead of `Role` for `roleRef`
  * Set the subject namespace to the namespace in which the operator is deployed.
* `deploy/service_account.yaml`:
  * Set `metadata.namespace` to the namespace where the operator is deployed.
 

### Example for cluster-scoped operator

With the above changes the specified manifests should look as follows:

* `deploy/operator.yaml`:
    ```YAML
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: memcached-operator
      namespace: <operator-namespace>
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
* `deploy/service_account.yaml`
    ```YAML
    apiVersion: v1
    kind: ServiceAccount
    metadata:
      name: memcached-operator
      namespace: <operator-namespace>
    ```
  
[RBAC]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/
[manager_user_guide]: /docs/golang/quickstart/#manager
[manager_options]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/manager#Options
[crd-scope-doc]: /docs/crds-scope