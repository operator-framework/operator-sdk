---
title: Ansible Operator Watches
linkTitle: Watches
weight: 20
---

The Watches file contains a list of mappings from custom resources, identified
by it's Group, Version, and Kind, to an Ansible Role or Playbook. The Operator
expects this mapping file in a predefined location: `/opt/ansible/watches.yaml`
These resources, as well as child resources (determined by owner references) will
be monitored for updates and cached.

* **group**:  The group of the Custom Resource that you will be watching.
* **version**:  The version of the Custom Resource that you will be watching.
* **kind**:  The kind of the Custom Resource that you will be watching.
* **role** (default): Specifies a role to be executed. This field is mutually exclusive with the
  "playbook" field. This field can be:
  * an absolute path to a role directory.
  * a relative path within one of the directories specified by `ANSIBLE_ROLES_PATH` environment variable or `ansible-roles-path` flag.
  * a relative path within the current working directory, which defaults to `/opt/ansible/roles`.
  * a fully qualified collection name of an installed Ansible collection. Ansible collections are installed to
    `~/.ansible/collections` or `/usr/share/ansible/collections` by default. If they are installed elsewhere,
    use the `ANSIBLE_COLLECTIONS_PATH` environment variable or the `ansible-collections-path` flag
* **playbook**: This is the playbook name that you have added to the
  container. This playbook is expected to be simply a way to call roles. This
  field is mutually exclusive with the "role" field. When running locally, the playbook is expected to be in the
  current project directory.
* **vars**: This is an arbitrary map of key-value pairs. The contents will be
  passed as `extra_vars` to the playbook or role specified for this watch.
* **reconcilePeriod** (optional): The maximum interval that the operator will wait before beginning another reconcile, even if no watched events are received. When an operator watches many resources, each reconcile can become expensive, and a low value here can actually reduce performance. Typically, this option should only be used in advanced use cases where `watchDependentResources` is set to `False`  and when is not possible to use the watch feature. E.g To manage external resources that don’t emit Kubernetes events. The format for the duration string is a sequence of decimal numbers, each with optional fraction and a unit suffix, such as "300ms", "1.5h" or "2h45m". Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h". Defaults to 10 hours. 
* **manageStatus** (optional): When true (default), the operator will manage
  the status of the CR generically. Set to false, the status of the CR is
  managed elsewhere, by the specified role/playbook or in a separate controller.
* **blacklist**: A list of child resources (by GVK) that will not be watched or cached.

An example Watches file:

```yaml
---
# Simple example mapping Foo to the Foo role
- version: v1alpha1
  group: foo.example.com
  kind: Foo
  role: Foo

# Simple example mapping Bar to a playbook
- version: v1alpha1
  group: bar.example.com
  kind: Bar
  playbook: playbook.yml

# More complex example for our Baz kind
# Here we will disable requeuing and be managing the CR status in the playbook,
# and specify additional variables.
- version: v1alpha1
  group: baz.example.com
  kind: Baz
  playbook: baz.yml
  manageStatus: False
  vars:
    foo: bar

# ConfigMaps owned by a Memcached CR will not be watched or cached.
- version: v1alpha1
  group: cache.example.com
  kind: Memcached
  role: /opt/ansible/roles/memcached
  blacklist:
    - group: ""
      version: v1
      kind: ConfigMap

# Example usage with a role from an installed Ansible collection
- version: v1alpha1
  group: bar.example.com
  kind: Bar
  role: myNamespace.myCollection.myRole

# Example filtering of resources with specific labels
- version: v1alpha1
  group: bar.example.com
  kind: Bar
  playbook: playbook.yml
  selector:
    matchLabels:
      foo: bar
    matchExpressions:
      - {key: foo, operator: In, values: [bar]}
      - {key: baz, operator: Exists, values: []}
```


The advanced features can be enabled by adding them to your watches file per GVK.
They can go below the `group`, `version`, `kind` and `playbook` or `role`.

Some features can be overridden per resource via an annotation on that CR. The options that are overridable will have the annotation specified below.

| Feature | Yaml Key | Description| Annotation for override | default | Documentation |
|---------|----------|------------|-------------------------|---------|---------------|
| Reconcile Period | `reconcilePeriod`  | time between reconcile runs for a particular CR  | ansible.sdk.operatorframework.io/reconcile-period  | 10h | |
| Manage Status | `manageStatus` | Allows the ansible operator to manage the conditions section of each resource's status section. | | true | |
| Watching Dependent Resources | `watchDependentResources` | Allows the ansible operator to dynamically watch resources that are created by ansible | | true | [dependent watches](../dependent-watches) |
| Watching Cluster-Scoped Resources | `watchClusterScopedResources` | Allows the ansible operator to watch cluster-scoped resources that are created by ansible | | false | |
| Max Runner Artifacts | `maxRunnerArtifacts` | Manages the number of [artifact directories](https://ansible-runner.readthedocs.io/en/latest/intro.html#runner-artifacts-directory-hierarchy) that ansible runner will keep in the operator container for each individual resource. | ansible.sdk.operatorframework.io/max-runner-artifacts | 20 | |
| Finalizer | `finalizer`  | Sets a finalizer on the CR and maps a deletion event to a playbook or role | | | [finalizers](../finalizers)|
| Selector | `selector`  | Identifies a set of objects based on their labels | | None Applied | [Labels and Selectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)|
| Automatic Case Conversion | `snakeCaseParameters`  | Determines whether to convert the CR spec from camelCase to snake_case before passing the contents to Ansible as extra_vars| | true | |
| Watching Annotations Changes| `watchAnnotationsChanges` | Allows the ansible operator to trigger reconciliations on annotations changes on watched resources | | false | |


#### Example
```YaML
---
- version: v1alpha1
  group: app.example.com
  kind: AppService
  playbook: playbook.yml
  maxRunnerArtifacts: 30
  reconcilePeriod: 5s
  manageStatus: False
  watchDependentResources: False
  snakeCaseParameters: False
  finalizer:
    name: app.example.com/finalizer
    vars:
      state: absent
```

**Note:** By using the command `operator-sdk add api` you are able to add additional CRDs to the project API, which can aid in designing your solution using concepts such as encapsulation, single responsibility principle, and cohesion, which could make the project easier to read, debug, and maintain. With this approach, you are able to customize and optimize the configurations more specifically per GVK via the `watches.yaml` file.

**Example:** 

```YaML
---
- version: v1alpha1
  group: app.example.com
  kind: AppService
  playbook: playbook.yml
  maxRunnerArtifacts: 30
  reconcilePeriod: 5s
  manageStatus: False
  watchDependentResources: False
  finalizer:
    name: app.example.com/finalizer
    vars:
      state: absent

- version: v1alpha1
  group: app.example.com
  kind: Database
  playbook: playbook.yml
  watchDependentResources: True
  manageStatus: True
```
