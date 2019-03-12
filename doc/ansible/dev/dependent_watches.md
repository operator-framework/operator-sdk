# Dependent Watches
This document describes the `watchDependentResources` option in [`watches.yaml`](#Example) file. It delves into what dependent resources are, why the option is required, how it is achieved and finally gives an example.

### What are dependent resources?
In most cases, an operator creates a bunch of Kubernetes resources in the cluster, that helps deploy and manage the application. For instance, the [etcd-operator](https://github.com/coreos/etcd-operator/blob/master/doc/gif/demo.gif) creates two services and a number of pods for a single `EtcdCluster` CR. In this case, all the Kubernetes resources created by the operator for a CR is defined as dependent resources.

### Why the `watchDependentResources` option?
Often, an operator needs to watch dependent resources. To achieve this, a developer would set the field, `watchDependentResources` to `True` in the `watches.yaml` file. If enabled, a change in a dependent resource will trigger the reconciliation loop causing Ansible code to run.

For example, since the _etcd-operator_ needs to ensure that all the pods are up and running, it needs to know when a pod changes. Enabling the dependent watches option would trigger the reconciliation loop to run. The Ansible logic needs to handle these cases and make sure that all the dependent resources are in the desired state as declared by the `CR spec`

`Note: By default it is enabled when using ansible-operator`

### How is this achieved?
The `ansible-operator` base image achieves this by leveraging the concept of [owner-references](https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/). Whenever a Kubernetes resource is created by Ansible code, the `ansible-operator`'s `proxy` module injects `owner-references` into the resource being created. The `owner-references` means the resource is owned by the CR for which reconciliation is taking place.

Whenever the `watchDependentResources` field is enabled, the `ansible-operator` will watch all the resources owned by the CR, registering callbacks to their change events. Upon a change, the callback will enqueue a `ReconcileRequest` for the CR. The enqueued reconciliation request will trigger the `Reconcile` function of the controller which will execute the ansible logic for reconciliation.

### Example

This is an example of a watches file with the `watchDependentResources` field set to `True`
```yaml

- version: v1alpha1
  group: app.example.com
  kind: AppService
  playbook: /opt/ansible/playbook.yml
  maxRunnerArtifacts: 30
  reconcilePeriod: 5s
  manageStatus: False
  watchDependentResources: True

```