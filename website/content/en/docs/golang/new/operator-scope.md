---
title: Operators Scope
linkTitle: Operator Scope
weight: 20
---

## Overview

A namespace-scoped operator watches and manages resources in a single Namespace, whereas a cluster-scoped operator
 watches and manages resources cluster-wide.

An operator should be cluster-scoped if it watches resources that can be created in any Namespace. An operator should 
be namespace-scoped if it is intended to be flexibly deployed. This scope permits 
decoupled upgrades, namespace isolation for failures and monitoring, and differing API definitions.

By default, `operator-sdk init` scaffolds a cluster-scoped operator. This document details conversion of default 
operator projects to namespaced-scoped operators. Before proceeding, be aware that your operator may be better suited 
as cluster-scoped. For example, the [cert-manager][cert-manager] operator is often deployed with cluster-scoped 
permissions and watches so that it can manage and issue certificates for an entire cluster.

**IMPORTANT**: When a [Manager][ctrl-manager] instance is created in the `main.go` file, the 
Namespaces are set via [Manager Options][ctrl-options] as described below. These Namespaces should be watched and 
cached for the Client which is provided by the Manager.Only clients provided by cluster-scoped Managers are able 
to manage cluster-scoped CRD's. For further information see: [CRD scope doc][crd-scope-doc].

## Watching resources in all Namespaces (default)

A [Manager][ctrl-manager] is initialized with no Namespace option specified, or `Namespace: ""` will 
watch all Namespaces:

```go
...
mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
    Scheme:             scheme,
    MetricsBindAddress: metricsAddr,
    Port:               9443,
    LeaderElection:     enableLeaderElection, 
    LeaderElectionID:   "f1c5ece8.example.com",
})
...
```

## Watching resources in a single Namespace

To restrict the scope of the [Manager's][ctrl-manager] cache to a specific Namespace set the `Namespace` field 
in [Options][ctrl-options]:

```go
...
mgr, err = ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
    Scheme:             scheme,
    Namespace:          "operator-namespace",
    MetricsBindAddress: metricsAddr,
})
...
``` 

## Watching resources in a set of Namespaces

It is possible to use [`MultiNamespacedCacheBuilder`][multi-namespaced-cache-builder] from 
[Options][ctrl-options] to watch and manage resources in a set of Namespaces:

```go
...
namespaces := []string{"foo", "bar"} // List of Namespaces

mgr, err = ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
    Scheme:             scheme,
    NewCache:           cache.MultiNamespacedCacheBuilder(namespaces),
    MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
})
...
```
In the above example, a CR created in a Namespace not in the set passed to `Options` will not be reconciled by 
its controller because the [Manager][ctrl-manager] does not manage that Namespace.

**IMPORTANT:** Note that this is not intended to be used for excluding Namespaces, this is better done via a Predicate. 

## Restricting Roles and permissions

An operator's scope defines its [Manager's][ctrl-manager] cache's scope but not the permissions to access the resources. 
After updating the Manager's scope to be Namespaced, the cluster's [Role-Based Access Control (RBAC)][k8s-rbac] 
permissions should be restricted accordingly.

These permissions are found in the directory `config/rbac/`. The `ClusterRole` in `role.yaml` and `ClusterRoleBinding` 
in `role_binding.yaml` are used to grant the operator permissions to access and manage its resources.

**NOTE** For changing the operator's scope only the `role.yaml` and `role_binding.yaml` manifests need to be updated. 
For the purposes of this doc, the other RBAC manifests `<kind>_editor_role.yaml`, `<kind>_viewer_role.yaml`, 
and `auth_proxy_*.yaml` are not relevant to changing the operator's resource permissions.

### Changing the permissions 

To change the scope of the RBAC permissions from cluster-wide to a specific namespace, you will need to use `Role`s 
and `RoleBinding`s instead of `ClusterRole`s and `ClusterRoleBinding`s, respectively.
 
[`RBAC markers`][rbac-markers] defined in the controller (e.g `controllers/memcached_controller.go`)
are used to generate the operator's [RBAC ClusterRole][rbac-clusterrole] (e.g `config/rbac/role.yaml`). The default
 markers don't specify a `namespace` property and will result in a `ClusterRole`.

Update the RBAC markers to specify a `namespace` property so that `config/rbac/role.yaml` is generated as a `Role`
 instead of a `ClusterRole`.

Replace:

```go
// +kubebuilder:rbac:groups=cache.example.com,resources=memcacheds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cache.example.com,resources=memcacheds/status,verbs=get;update;patch
```

With namespaced markers:

```go
// +kubebuilder:rbac:groups=cache.example.com,namespace="my-namespace",resources=memcacheds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cache.example.com,namespace="my-namespace",resources=memcacheds/status,verbs=get;update;patch
```

And then, run `make manifests` to update `config/rbac/role.yaml`:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
...
```

We also need to update our `ClusterRoleBindings` to `RoleBindings` since they are not regenerated: 

```yaml
  
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: manager-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: manager-role
subjects:
- kind: ServiceAccount
  name: default
  namespace: system
```

<!-- todo(camilamacedo86): The need for the RoleBinding show an issue tracked 
in https://github.com/kubernetes-sigs/kubebuilder/issues/1496. --> 

## Using environment variables for Namespace 

Instead of having any Namespaces hard-coded in the `main.go` file a good practice is to use environment 
variables to allow the restrictive configurations

### Configuring Namespace scoped operators

- Add a helper function in the `main.go` file:

```go
// getWatchNamespace returns the Namespace the operator should be watching for changes
func getWatchNamespace() (string, error) {
    // WatchNamespaceEnvVar is the constant for env variable WATCH_NAMESPACE
    // which specifies the Namespace to watch.
    // An empty value means the operator is running with cluster scope.
    var watchNamespaceEnvVar = "WATCH_NAMESPACE"
    
    ns, found := os.LookupEnv(watchNamespaceEnvVar)
    if !found {
        return "", fmt.Errorf("%s must be set", watchNamespaceEnvVar)
    }
    return ns, nil
}
```

- Use the environment variable value: 

```go
...
watchNamespace, err := getWatchNamespace()
if err != nil {
    setupLog.Error(err, "unable to get WatchNamespace, " +
       "the manager will watch and manage resources in all namespaces")
}

mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
    Scheme:             scheme,
    MetricsBindAddress: metricsAddr,
    Port:               9443,
    LeaderElection:     enableLeaderElection,
    LeaderElectionID:   "f1c5ece8.example.com",
    Namespace:          watchNamespace, // namespaced-scope when the value is not an empty string 
})
...
```

- Define the environment variable in the `config/manager/manager.yaml`:

```yaml
spec:
  containers:
  - command:
    - /manager
    args:
    - --enable-leader-election
    image: controller:latest
    name: manager
    resources:
      limits:
        cpu: 100m
        memory: 30Mi
      requests:
        cpu: 100m
        memory: 20Mi
    env:
      - name: WATCH_NAMESPACE
        valueFrom:
          fieldRef:
            fieldPath: metadata.namespace 
  terminationGracePeriodSeconds: 10
``` 

**NOTE** `WATCH_NAMESPACE` here will always be set as the namespace where the operator is deployed. 

### Configuring cluster-scoped operators with MultiNamespacedCacheBuilder

- Add a helper function to get the environment variable value in the `main.go` file as done in the previous example (e.g `getWatchNamespace()`)
- Use the environment variable value and check if it is a multi-namespace scenario:

```go
    ...
watchNamespace, err := getWatchNamespace()
if err != nil {
    setupLog.Error(err, "unable to get WatchNamespace, " +
        "the manager will watch and manage resources in all Namespaces")
}

options := ctrl.Options{
    Scheme:             scheme,
    MetricsBindAddress: metricsAddr,
    Port:               9443,
    LeaderElection:     enableLeaderElection,
    LeaderElectionID:   "f1c5ece8.example.com",
    Namespace:          watchNamespace, // namespaced-scope when the value is not an empty string 
}

// Add support for MultiNamespace set in WATCH_NAMESPACE (e.g ns1,ns2)
if strings.Contains(namespace, ",") {
    setupLog.Infof("manager will be watching namespace %q", watchNamespace) 
    // configure cluster-scoped with MultiNamespacedCacheBuilder
    options.Namespace = ""
    options.NewCache = cache.MultiNamespacedCacheBuilder(strings.Split(watchNamespace, ","))
}
...
```

- Define the environment variable in the `config/manager/manager.yaml`:

```yaml
...
    env:
      - name: WATCH_NAMESPACE
        value: "ns1,ns2"
  terminationGracePeriodSeconds: 10
...
``` 

[cert-manager]: https://github.com/jetstack/cert-manager
[ctrl-manager]: https://godoc.org/sigs.k8s.io/controller-runtime/pkg/manager#Manager
[ctrl-options]: https://godoc.org/sigs.k8s.io/controller-runtime/pkg/manager#Options
[multi-namespaced-cache-builder]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/cache#MultiNamespacedCacheBuilder
[k8s-rbac]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/
[kube-rbac-proxy]: https://github.com/brancz/kube-rbac-proxy
[rbac-clusterrole]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/#role-and-clusterrole
[crd-scope-doc]: /docs/golang/new/crds-scope/
[rbac-markers]: https://book.kubebuilder.io/reference/markers/rbac.html