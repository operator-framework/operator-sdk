## Advanced Options

This document shows the advanced options available to a developer of an ansible operator.

### Watches File Options

The advanced features can be enabled by adding them to your watches file per GVK.
They can go below the `group`, `version`, `kind` and `playbook` or `role`.

Some features can be overridden per resource via an annotation on that CR. The options that are overridable will have the annotation specified below.

| Feature | Yaml Key | Description| Annotation for override | default | Documentation |
|---------|----------|------------|-------------------------|---------|---------------|
| Reconcile Period | `reconcilePeriod`  | time between reconcile runs for a particular CR  | ansbile.operator-sdk/reconcile-period  | 1m | |
| Manage Status | `manageStatus` | Allows the ansible operator to manage the conditions section of each resource's status section. | | true | |
| Watching Dependent Resources | `watchDependentResources` | Allows the ansible operator to dynamically watch resources that are created by ansible | | true | [dependent_watches.md](dependent_watches.md) |
| Watching Cluster-Scoped Resources | `watchClusterScopedResources` | Allows the ansible operator to watch cluster-scoped resources that are created by ansible | | false | |
| Max Runner Artifacts | `maxRunnerArtifacts` | Manages the number of [artifact directories](https://ansible-runner.readthedocs.io/en/latest/intro.html#runner-artifacts-directory-hierarchy) that ansible runner will keep in the operator container for each individual resource. | ansible.operator-sdk/max-runner-artifacts | 20 | |
| Finalizer | `finalizer`  | Sets a finalizer on the CR and maps a deletion event to a playbook or role | | | [finalizers.md](finalizers.md)|


#### Example
```YaML
---
- version: v1alpha1
  group: app.example.com
  kind: AppService
  playbook: /opt/ansible/playbook.yml
  maxRunnerArtifacts: 30
  reconcilePeriod: 5s
  manageStatus: False
  watchDependentResources: False
  finalizer:
    name: finalizer.app.example.com
    vars:
      state: absent
```


### Runner Directory

The ansible runner will keep information about the ansible run in the container.  This is located `/tmp/ansible-operator/runner/<group>/<version>/<kind>/<namespace>/<name>`. To learn more  about the runner directory you can read the [ansible-runner docs](https://ansible-runner.readthedocs.io/en/latest/index.html).

## Turning off Dependent Watches and Owner Reference Injection

You may want to manage what your operator watches and the owner references. This means that your operator will need to understand how to clean up after itself when your CR is deleted. To disable these features you will need to edit your `build/Dockerfile` to include the line below.

**NOTE**: That if you use this feature there will be a warning that dependent watches is turned off but there will be no error.

```
ENTRYPOINT ["/usr/local/bin/entrypoint", "--inject-owner-ref=false"]
```
