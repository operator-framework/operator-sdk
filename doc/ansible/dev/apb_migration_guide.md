# Migrating an APB to an Ansible Operator

## Basic process
### Directory structure and metadata
#### Generate Operator resources
Generate an operator with
```
operator-sdk new <name> --type=ansible --api-version=<group>/<version> --kind=<kind>`
```
where:

* `<name>` is the name for your operator, so if for example you have a `memcached-apb`, you would probably use `memcached-operator`
* `<group>` is the API group for your Kubernetes Custom Resource Definition. For example, if I own the domain `example.com`, I might use the group `apps.example.com`.
* `<version>` is the API version for your Kubernete Custom Resource Definition. `v1alpha1` is a common starting value, with `v1beta1` implying a fair amount of API stability and `v1` implying no breaking API changes at all.
* `<kind>` is the kind of your resource. For example, if you are creating a `memcached-operator`, your `kind` would likely be `Memcached`

So for the example `memcached-operator`, the command would be:

```
operator-sdk new memcached-operator --type=ansible --api-version=apps.example.com/v1alpha1 --kind=Memcached
```


Once this is generated, take the `build`, `deploy`, and `molecule` directories, as well as the `watches.yaml` and copy them into your APB directory.

#### Dockerfile
You now have two Dockerfiles, your original APB `Dockerfile` at the top-level, and a `build/Dockerfile` for your operator. 

In your `build/Dockerfile`, ensure that your playbooks and roles are being copied to `${HOME}/roles` and `${HOME}/playbooks`, and that your `watches.yaml` is being copied to `${HOME}/watches.yaml`. 

If you are installing any additional dependencies, ensure that those are reflected in your `build/Dockerfile` as well. 

As a sample, your `build/Dockerfile` will probably look something like this:

```
FROM quay.io/operator-framework/ansible-operator:v0.9.0

COPY watches.yaml ${HOME}/watches.yaml

COPY roles/ ${HOME}/roles/
COPY playbooks/ ${HOME}/playbooks/
```

Once this is done you may remove your original APB `Dockerfile`.

#### watches.yaml
In the `watches.yaml`, ensure the playbook for your `kind` points to your `provision.yml` playbook in the container (likely location for that will be `/opt/ansible/playbooks/provision.yml`). 

Next, add a finalizer block with a name of: `finalizer.<name>.<group>/<version>`, and set the playbook to point to your `deprovision.yml` in the container (likely location for that will be `/opt/ansible/playbooks/deprovision.yml`). For the memcached-operator we generated above, the watches.yaml would look like this:

```yaml
---
- version: v1alpha1
  group: apps.example.com
  kind: Memcached
  playbook: /opt/ansible/playbooks/provision.yml
  finalizer:
    name: finalizer.memcached.apps.example.com/v1alpha1
    playbook: /opt/ansible/playbooks/deprovision.yml
```

##### Binding
If you have a `bind` playbook, add a new entry to your `watches.yaml` (you can copy paste the existing entry). 

The `version` and `group`, will remain unchanged, but update the `kind` with a `Binding` suffix. 

For example, if you have a resource with `kind: Memcached`, the kind of your new entry will be `MemcachedBinding`. 

The playbook for this entry should map to your `bind` playbook, (likely location `/opt/ansible/playbooks/bind.yml`), and if you have an `unbind` playbook then set the playbook for the finalizer to point to it (likely location `/opt/ansible/playbooks/unbind.yml`). If you don't have an `unbind` playbook, remove the finalizer block for your `Binding` resource.

For an APB with both `bind` and `unbind` playbooks, the `watches.yaml` would end up looking like this:

```yaml
---
- version: v1alpha1
  group: apps.example.com
  kind: Memcached
  playbook: /opt/ansible/playbooks/provision.yml
  finalizer:
    name: finalizer.memcached.apps.example.com/v1alpha1
    playbook: /opt/ansible/playbooks/deprovision.yml
- version: v1alpha1
  group: apps.example.com
  kind: MemcachedBinding
  playbook: /opt/ansible/playbooks/bind.yml
  finalizer:
    name: finalizer.memcachedbinding.apps.example.com/v1alpha1
    playbook: /opt/ansible/playbooks/unbind.yml
```


You will also need to run `operator-sdk add crd --api-version=<group>/<version> --kind=<kind>` to generate a new CRD and example in `deploy/crds`.

#### deploy/crds/
Now that you have all your CRDs created, you can generate the OpenAPI spec for them using your `apb.yml`. 

The `convert.py` script included at the bottom of this document can handle the conversion to the OpenAPI spec, at which point you can copy paste everything from `validation:` on into your primary CRD (for the regular `parameters`), or into your `Binding` CRD (for `bind_parameters`).

You may notice that the OpenAPI validation uses `camelCase` parameters, while your `apb.yml` and Ansible playbooks probably assume `snake_case` variables. `Ansible Operator` will automatically convert the `camelCase` parameters from the Kubernetes resource into `snake_case` before passing them to your playbook, so this should not require any change on your part.

### Ansible logic
There will be some changes required to your Ansible playbooks/roles/tasks.

#### Idempotence

One major conceptual difference between the APB model and the Operator model, is that APBs are meant to run `provision` once, while operators constantly reconcile to ensure that the state of the cluster matches the state that the user requested. 

This means that you will need to ensure that your playbooks are idempotent, and can be run repeatedly with the same parameters without causing an error.

#### Service Bundle contract and meta variables
Ansible Operator does not respect the Service Bundle contract that exists between APBs and the Ansible Service Broker. The following variables will not be passed in by the Ansible Operator:

- `cluster`: Operators ideally work on both Kubernetes and OpenShift, so any uses of openshift-specific resources should handle errors and fallback
- `_apb_plan_id`: Operators have no concept of a plan
- `_apb_service_class_id`: This concept is replaced by the group/version/kind specified in your CRD
- `_apb_service_instance_id`: This concept is replaced by `meta.name`, the name of the Custom Resource created by the user requesting the action.
- `_apb_last_requesting_user`: There is no analogue to this.
- `_apb_provision_creds`: There is no analogue to this.
- `_apb_service_binding_id`: This concept is replaced by the `meta.name` of a `<kind>Binding` resource
- `namespace`: This is accessible via the `meta.namespace` variable

Instead, the Ansible Operator will pass in a field called `meta`, which contains the `name` and `namespace` of the Custom Resource that the user created.


#### asb_encode_binding
This module will not be present in the Ansible Operator base image. In order to save credentials after a successful provision, you will need to create a `secret` in Kubernetes, and update the status of your custom resource so that people can find it. For example, if we have the following Custom Resource group/version/kind:

```yaml
version: v1alpha1
group: apps.example.com
kind: PostgreSQL
```

the following task:

```yaml
- name: encode bind credentials
  asb_encode_binding:
    fields:
      DB_TYPE: postgres
      DB_HOST: "{{ app_name }}"
      DB_PORT: "5432"
      DB_USER: "{{ postgresql_user }}"
      DB_PASSWORD: "{{ postgresql_password }}"
      DB_NAME: "{{ postgresql_database }}"
```

would become:

```yaml
- name: Create bind credential secret
  k8s:
    definition:
      apiVersion: v1
      kind: Secret
      metadata:
        name: '{{ meta.name }}-credentials'
        namespace: '{{ meta.namespace }}'
      data:
        DB_TYPE: "{{ 'postgres' | b64encode }}"
        DB_HOST: "{{ app_name | b64encode }}"
        DB_PORT: "{{ '5432' | b64encode }}"
        DB_USER: "{{ postgresql_user | b64encode }}"
        DB_PASSWORD: "{{ postgresql_password | b64encode }}"
        DB_NAME: "{{ postgresql_database | b64encode }}"

- name: Attach secret to CR status
  k8s_status:
    api_version: apps.example.com/v1alpha1
    kind: PostgreSQL
    name: '{{ meta.name }}'
    namespace: '{{ meta.namespace }}'
    status:
      bind_credentials_secret: '{{ meta.name }}-credentials'
```

#### ansible_kubernetes_modules
* The ansible_kubernetes_modules role and the generated modules are now deprecated.
* The `k8s` module was added in Ansible 2.6 and is the supported way to interact with Kubernetes from Ansible.
* The `k8s` module takes normal kubernetes manifests, so if you currently rely on the old generated modules some refactoring will be required.


# Terms
kind
group
apiVersion
watches.yaml
finalizer
CRD

# convert.py
This script should be run from inside the APB directory, next to the `apb.yml`
```python
#!/usr/bin/env python

import yaml


def extract_params(all_params):
    properties = {}
    required = set()
    for param in all_params:
        name = param['name']
        name_parts = name.split('_')
        camel_name = name_parts[0] + ''.join([x.title() for x in name_parts[1:]])
        if param.get('required') is True:
            if camel_name not in properties:
                required.add(camel_name)
        elif camel_name in required and param.get('required') is False:
            required.remove(camel_name)
        properties[camel_name] = {
            "type": param["type"],
            "description": param.get("description", param.get("title", ""))
        }

    return {
        "validation": {"openAPIv3Schema": {
            "properties": {
                "spec": {
                    "required": list(required),
                    "properties": properties
                }
            }
        }}
    }


def main():
    with open('apb.yml', 'r') as f:
        apb_meta = yaml.safe_load(f.read())

    for field in ['parameters', 'bind_parameters']:
        print("Converting {0} to OpenAPI spec".format(field))
        print(yaml.dump({field: extract_params([
            param for x in apb_meta['plans'] for param in x.get(field, [])
        ])}))


if __name__ == '__main__':
    main()
```
