---
title: Ansible Operator Advanced Options
linkTitle: Advanced Options
weight: 20
---

This document shows the advanced options available to a developer of an ansible operator.

### Runner Directory

The ansible runner will keep information about the ansible run in the container.  This is located `/tmp/ansible-operator/runner/<group>/<version>/<kind>/<namespace>/<name>`. To learn more  about the runner directory you can read the [ansible-runner docs](https://ansible-runner.readthedocs.io/en/latest/index.html).

## Owner Reference Injection

Owner references enable [Kubernetes Garbage Collection](https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/) to clean up after a CR is deleted. Owner references are injected by ansible operators by default by the proxy.

Owner references only apply to resources in the same namespace as the CR. Resources outside the namespace of the CR will automatically be annotated with `operator-sdk/primary-resource` and `operator-sdk/primary-resource-type` to track creation. These resources will not be automatically garbage collected. To handle deletion of these resources, use a [finalizer](../finalizers).

You may want to manage what your operator watches and the owner references. This means that your operator will need to understand how to clean up after itself when your CR is deleted. To disable these features you will need to edit your `build/Dockerfile` to include the line below.

**NOTE**: That if you use this feature there will be a warning that dependent watches is turned off but there will be no error.
**WARNING**: Once a CR is deployed without owner reference injection, there is no automatic way to add those references.

```
ENTRYPOINT ["/usr/local/bin/entrypoint", "--inject-owner-ref=false"]
```

If you have created resources without owner reference injection, it is
possible to manually to update resources following [this
guide.](../retroactively-owned-resources)

## Max Workers

Increasing the number of workers allows events to be processed
concurrently, which can improve reconciliation performance.

Worker maximums can be set in two ways. Operator **authors and admins**
can set the max workers default by including extra args to the operator
container in `deploy/operator.yaml`. (Otherwise, the default is 1 worker.)

**NOTE:** Admins using OLM should use the environment variable instead
of the extra args.

``` yaml
- name: operator
  image: "quay.io/asmacdo/memcached-operator:v0.0.0"
  imagePullPolicy: "Always"
  args:
    - "--max-workers"
    - "3"
```

Operator **admins** can override the value by setting an environment
variable in the format `WORKER_<kind>_<group>`. This variable must be
all uppercase, and periods (e.g. in the group name) are replaced with underscores.

For the memcached operator example, the component parts are retrieved
with a GET on the operator:

```bash
$ kubectl get memcacheds example-memcached -o yaml

apiVersion: cache.example.com/v1alpha1
kind: Memcached
metadata:
  name: example-memcached
  namespace: default
```

From this data, we can see that the environment variable will be
`WORKER_MEMCACHED_CACHE_EXAMPLE_COM`, which we can then add to
`deploy/operator.yaml`:

``` yaml
- name: operator
  image: "quay.io/asmacdo/memcached-operator:v0.0.0"
  imagePullPolicy: "Always"
  args:
    # This default is overridden.
    - "--max-workers"
    - "3"
  env:
    # This value is used
    - name: WORKER_MEMCACHED_CACHE_EXAMPLE_COM
      value: "6"
```

## Ansible Verbosity

Setting the verbosity at which `ansible-runner` is run controls how verbose the
output of `ansible-playbook` will be. The normal rules for verbosity apply
here, where higher values mean more output. Acceptable values range from 0
(only the most severe messages are output) to 7 (all debugging messages are
output).

There are three ways to configure the verbosity argument to the `ansible-runner`
command:

1. Operator **authors and admins** can set the Ansible verbosity by including
   extra args to the operator container in the operator deployment.
1. Operator **admins** can set Ansible verbosity by setting an environment
   variable in the format `ANSIBLE_VERBOSITY_<kind>_<group>`. This variable must
   be all uppercase and all periods (e.g. in the group name) are replaced with
   underscore.
1. Operator **users, authors, and admins** can set the Ansible verbosity by
   setting the `"ansible.operator-sdk/verbosity"` annotation on the Custom
   Resource.

### Examples

For demonstration purposes, let us assume that we have a database operator that
supports two Kinds -- `MongoDB` and `PostgreSQL` -- in the `db.example.com`
Group. We have only recently implemented the support for the `MongoDB` Kind so
we want reconciles for this Kind to be more verbose. Our operator container's
spec in our `deploy/operator.yaml` might look something like:

```yaml
- name: operator
  image: "quay.io/example/database-operator:v1.0.0"
  imagePullPolicy: "Always"
  args:
    # This value applies to all GVKs specified in watches.yaml
    # that are not overriden by environment variables.
    - "--ansible-verbosity"
    - "1"
  env:
    # Override the verbosity for the MongoDB kind
    - name: ANSIBLE_VERBOSITY_MONGODB_DB_EXAMPLE_COM
      value: "4"
```

Once the Operator is deployed, the only way to change the verbosity is via the
`"ansible.operator-sdk/verbosity"` annotation. Continuing with our example, our
CR may look like:

```yaml
apiVersion: "db.example.com/v1"
kind: "PostgreSQL"
metadata:
  name: "example-db"
  annotations:
    "ansible.operator-sdk/verbosity": "5"
spec: {}
```

## Custom Resources with OpenApi Validation

Currently, SDK tool does not support and will not generate automatically the CRD's using the [OpenAPI](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#validation) spec to perform validations. 

However, it can be done manually by adding its validations as you can check in the following example.

**Example**

```yaml
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
  subresources:
    status: {}
  validation:
    openAPIV3Schema:
      description: Memcached is the Schema for the memcacheds API
      properties:
        apiVersion:
          description: 'APIVersion defines the versioned schema of this representation
            of an object. Servers should convert recognized schemas to the latest
            internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
          type: string
        kind:
          description: 'Kind is a string value representing the REST resource this
            object represents. Servers may infer this from the endpoint the client
            submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
          type: string
        metadata:
          type: object
        spec:
          description: MemcachedSpec defines the desired state of Memcached
          properties:
            size:
              description: Size is the size of the memcached deployment
              format: int32
              type: integer
          required:
          - size
          type: object
        status:
          description: MemcachedStatus defines the observed state of Memcached
          properties:
            nodes:
              description: Nodes are the names of the memcached pods
              items:
                type: string
              type: array
          required:
          - nodes
          type: object
      type: object
  versions:
  - name: v1alpha1
    served: true
    storage: true
```
