# Adding Owner References for Existing Resources

Owner references are automatically injected `only during creation of
resources`. Enabling owner reference injection `will not update objects`
created while [owner reference injection is
disabled](./advanced_options.md#turning-off-dependent-watches-and-owner-reference-injection)

This guide will demonstrate how to retroactively set owner references
for existing resources.

A GET request to the owning resource will provide the necessary data to
construct an `ownerReference` or an `annotation`.

`$ kubectl get memcacheds.cache.example.com -o yaml`

**Example Response (Abbreviated):**

```yaml
apiVersion: cache.example.com/v1alpha1
kind: Memcached
metadata:
  name: example-memcached
  namespace: default
  uid: 2a94ff2b-84e0-40ce-8b5e-2b7e4d2bc0e2
```

`kubectl edit` can be used to update the resources by hand. See below
for example `ownerReference` and `annotations`.

## For objects in the same namespace as the Owner (CR)

Dependent resources *within the same namespace as the owning CR* are
tracked with the `ownerReference` field.

`ownerReference` structure:
  * apiVersion: {group}/{version}
  * kind: {kind}
  * name: {metadata.name}
  * uid: {metadata.uid}

**Example ownerReference:**

```yaml
metadata:
  ...(snip)
  ownerReferences:
    - apiVersion: cache.example.com/v1alpha1
      kind: Memcached
      name: example-memcached
      uid: ad834522-d9a5-4841-beac-991ff3798c00
```

## For objects which are NOT in the same namespace as the Owner (CR)

An `annotation` is used instead of an `ownerReference` if the dependent
resource is in a different namespace than the CR, or the dependent
resource is a cluster level resource.

`annotation` structure:
  * operator-sdk/primary-resource: {metadata.namespace}/{metadata.name}
  * operator-sdk/primary-resource-type: {kind}.{group}

**NOTE**: The {group} can be found by splitting the `apiVersion`
metadata of the CR, into `group` and `version`. As an example, 
[this apiVersion field](https://github.com/operator-framework/operator-sdk-samples/blob/master/ansible/memcached-operator/deploy/crds/cache.example.com_v1alpha1_memcached_cr.yaml#L1)
gives us the group `cache.example.com`.

**Example Annotation:**

```yaml
metadata:
  ...(snip)
  annotations:
    operator-sdk/primary-resource: default/example-memcached
    operator-sdk/primary-resource-type: Memcached.cache.example.com
```

## Migration using Ansible assets 

If you have many resources to update, it may be easier to use the
following Ansible assets, which **should be considered an example rather
than an officially supported workflow**.

To use these assets, create a `vars.yml` as specified below and copy
`playbook.yml` and `each_resource.yml` into the same directory. Execute
the playbook with:

``` bash
$ ansible-playbook -i localhost playbook.yml
```

### vars.yml

This file should be created by the user to configure the playbook, and
needs to contain:

  * owning_resource
      * apiVersion
      * kind
      * name
      * namespace
  * resources_to_own (list): For each resource, specify:
      * name
      * namespace (if applicable)
      * apiVersion
      * kind

```yaml
owning_resource:
  apiVersion: cache.example.com/v1alpha1
  kind: Memcached
  name: example-memcached
  namespace: default

resources_to_own:
  - name: example-memcached-memcached
    namespace: default
    apiVersion: apps/v1
    kind: Deployment
  - name: example-memcached
    apiVersion: v1
    kind: Namespace
```

### playbook.yml

This file can be used as-is without user adjustments.

```yaml
- hosts: localhost

  tasks:
    - name: Import user variables
      include_vars: vars.yml
    - name: Retrieve owning resource
      k8s_info:
        api_version: "{{ owning_resource.apiVersion }}"
        kind: "{{ owning_resource.kind }}"
        name: "{{ owning_resource.name }}"
        namespace: "{{ owning_resource.namespace }}"
      register: extra_owner_data

    - name: Ensure resources are owned
      include_tasks: each_resource.yml
      loop: "{{ resources_to_own }}"
      vars:
        to_be_owned: '{{ q("k8s",
          api_version=item.apiVersion,
          kind=item.kind,
          resource_name=item.name,
          namespace=item.namespace
        ).0 }}'
        owner_reference:
          apiVersion: "{{ owning_resource.apiVersion }}"
          kind: "{{ owning_resource.kind }}"
          name: "{{ owning_resource.name }}"
          uid: "{{ extra_owner_data.resources[0].metadata.uid }}"
```

### `each_resource.yml`

This file can be used as-is without user adjustments.

``` yaml
- name: Patch resource with owner reference
  when:
    - to_be_owned.metadata.namespace is defined
    - to_be_owned.metadata.namespace == owning_resource.namespace
    - (to_be_owned.metadata.ownerReferences is not defined) or
      (owner_reference not in to_be_owned.metadata.ownerReferences)
  k8s:
    state: present
    resource_definition:
      apiVersion: "{{ to_be_owned.apiVersion }}"
      kind: "{{ to_be_owned.kind }}"
      metadata:
        name: "{{ to_be_owned.metadata.name }}"
        namespace: "{{ to_be_owned.metadata.namespace }}"
        ownerReferences: "{{ (to_be_owned.metadata.ownerReferences | default([])) + [owner_reference] }}"

- name: Patch resource with owner annotation
  when: to_be_owned.namespace is not defined or to_be_owned.namespace != owning_resource.namespace
  k8s:
    state: present
    resource_definition:
      apiVersion: "{{ to_be_owned.apiVersion }}"
      kind: "{{ to_be_owned.kind }}"
      metadata:
        name: "{{ to_be_owned.metadata.name }}"
        namespace: "{{ to_be_owned.metadata.namespace | default(omit)}}"
        annotations:
          operator-sdk/primary-resource: "{{ owning_resource.namespace }}/{{ owning_resource.name }}"
          operator-sdk/primary-resource-type: "{{ owning_resource.kind }}.{{ owning_resource.apiVersion.split('/')[0] }}"
```
