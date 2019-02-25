# Handling deletion events

The default behavior of an Ansible Operator is to delete all resources the operator
created during reconciliation when a managed resource is marked for deletion. This
behavior is usually sufficient for applications that exist only in Kubernetes, but
sometimes it is necessary to perform more complex operations (for example, when
your action performed against a third party API needs to be undone). These more
complex cases can still be handled by Ansible Operator, through the use of a finalizer.

Finalizers allow controllers (such as an Ansible Operator) to implement asynchronous pre-delete hooks.
This allows custom logic to run after a resource has been marked for deletion, but
before the resource has actually been deleted from the Kubernetes cluster.
For Ansible Operator, this hook takes the form of an Ansible playbook or role. You can
define the mapping from your finalizer to a playbook or role by simply setting the
`finalizer` field on the entry in your `watches.yaml`. You can also choose to re-run
your top-level playbook or role with different variables set. The `watches.yaml`
finalizer configuration accepts the following options:


#### name
`name` is required.

This is the name of the finalizer. This is basically an arbitrary string, the existence
of any finalizer string on a resource will prevent that resource from being deleted until
the finalizer is removed. Ansible Operator will remove this string from the list of
finalizers on successful execution of the specified role or playbook. A typical finalizer
will be `finalizer.<group>`, where `<group>` is the group of the resource being managed.

#### playbook

One of `playbook`, `role`, or `vars` must be provided. If `playbook` is not provided, it
will default to the playbook specified at the top level of the `watches.yaml`
entry.

This field is identical to the top-level `playbook` field. It requires an absolute
path to a playbook on the operator’s file system.

#### role

One of `playbook`, `role`, or `vars` must be provided. If `role` is not provided, it
will default to the role specified at the top level of the `watches.yaml` entry.

This field is identical to the top-level `role` field. It requires an absolute
path to a role on the operator’s file system.

#### vars

One of `playbook`, `role`, or `vars` must be provided.

`vars` is an arbitrary map of key-value pairs. The contents of `vars` will be passed as `extra_vars` to the
playbook or role specified in the finalizer block, or at the top-level if neither `playbook`
or `role` was set for the finalizer.

## Examples

Here are a few examples of `watches.yaml` files that specify a finalizer:

### Run top-level playbook or role with new variables
```yaml
---
- version: v1alpha1
  group: app.example.com
  kind: Database
  playbook: /opt/ansible/playbook.yml
  finalizer:
    name: finalizer.app.example.com
    vars:
      state: absent
```

This example will run `/opt/ansible/playbook.yml` when the Custom Resource
is deleted. Because `vars` is set, the playbook will be run with `state` set to `absent`. Inside the playbook,
the author can check this value and perform whatever cleanup is necessary.

```yaml
---
- version: v1alpha1
  group: app.example.com
  kind: Database
  role: /opt/ansible/roles/database
  finalizer:
    name: finalizer.app.example.com
    vars:
      state: absent
```

This example is nearly identical to the first, except it will run the `/opt/ansible/roles/database`
role, rather than a playbook, with the `state` variable set to `absent`.

### Run a different playbook or role
```yaml
---
- version: v1alpha1
  group: app.example.com
  kind: Database
  playbook: /opt/ansible/playbook.yml
  finalizer:
    name: finalizer.app.example.com
    role: /opt/ansible/roles/teardown_database
```

This example will run the `/opt/ansible/roles/teardown_database` role when the Custom Resource is deleted.

```yaml
---
- version: v1alpha1
  group: app.example.com
  kind: Database
  playbook: /opt/ansible/playbook.yml
  finalizer:
    name: finalizer.app.example.com
    playbook: /opt/ansible/destroy.yml
```

This example will run the `/opt/ansible/destroy.yml` playbook when the Custom Resource is deleted.

### Run a different playbook or role with vars

You can set `playbook` or `role` and `vars` at the same time. This can be useful if only a small
part of your logic handles interacting with the component that requires cleanup. Rather than
run all the logic again, you can specify only to run the role or playbook that handled the
interaction, with a different variable set.

```yaml
---
- version: v1alpha1
  group: app.example.com
  kind: Database
  playbook: /opt/ansible/playbook.yml
  finalizer:
    name: finalizer.app.example.com
    role: /opt/ansible/roles/manage_credentials
    vars:
      state: revoked
```


For this example, assume our application configures automated backups to a third party service.
On deletion, all we want to do is revoke the credentials used to backup the data. We run
just the `/opt/ansible/roles/manage_credentials` role, which is imported by our playbook to
create the credentials in the first place, but we pass the `state: revoked` option, which
causes the role to invalidate our credentials. For everything else in our application,
automatic deletion of dependent resources will be sufficient, so we can exit successfully and
let the operator remove our finalizer and allow the resource to be deleted.

## Further reading
• [Kubernetes finalizers](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#finalizers)
