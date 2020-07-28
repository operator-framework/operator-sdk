---
title: Ansible Operator Tutorial
linkTitle: Tutorial
weight: 3
---

This guide walks through an example of building a simple memcached-operator powered by Ansible using tools and libraries provided by the Operator SDK.

## Create a new project

After [installing the Operator SDK CLI][install-guide] and
[ansible operator prerequisites][ansible-install-guide], use the CLI to create a
new Ansible-based memcached-operator project:

```sh
$ mkdir memcached-operator && cd memcached-operator
$ operator-sdk init --plugins=ansible --domain=example.com
$ operator sdk create api --group=cache --version=v1alpha1 --kind=Memcached --generate-role
```

This creates the memcached-operator project specifically for watching the
Memcached resource with APIVersion `cache.example.com/v1alpha1` and Kind
`Memcached`.

To learn more about the project directory structure, see [project
layout][layout-doc] doc.

#### Operator scope

Read the [operator scope][operator-scope] documentation on how to run your operator as namespace-scoped vs cluster-scoped.

## Customize the operator logic

For this example the memcached-operator will execute the following
reconciliation logic for each `Memcached` Custom Resource (CR):
- Create a memcached Deployment if it doesn't exist
- Ensure that the Deployment size is the same as specified by the `Memcached`
CR

## Watch the Memcached CR

By default, the memcached-operator watches `Memcached` resource events as shown
in `watches.yaml` and executes Ansible Role `Memcached`:

```yaml
---
- version: v1alpha1
  group: cache.example.com
  kind: Memcached
```


#### Options

**Role**
Specifying a `role` option in `watches.yaml` will configure the operator to use
this specified path when launching `ansible-runner` with an Ansible Role. By
default, the `new` command will fill in an absolute path to where your role
should go.
```yaml
---
- version: v1alpha1
  group: cache.example.com
  kind: Memcached
  role: memcached
```

**Playbook**
Specifying a `playbook` option in `watches.yaml` will configure the operator to
use this specified path when launching `ansible-runner` with an Ansible
Playbook
```yaml
---
- version: v1alpha1
  group: cache.example.com
  kind: Memcached
  playbook: playbook.yaml
```

See [watches reference][ansible-watches] for more information.

## Building the Memcached Ansible Role

The first thing to do is to modify the generated Ansible role under
`roles/memcached`. This Ansible Role controls the logic that is executed when a
resource is modified.

### Define the Memcached spec

Defining the spec for an Ansible Operator can be done entirely in Ansible. The
Ansible Operator will simply pass all key value pairs listed in the Custom
Resource spec field along to Ansible as extra
[variables](https://docs.ansible.com/ansible/2.5/user_guide/playbooks_variables.html#passing-variables-on-the-command-line).
The names of all variables in the spec field are converted to snake_case
by the operator before running ansible. For example, `serviceAccount` in
the spec becomes `service_account` in ansible. You can disable this case conversion
by setting the `snakeCaseParameters` option to `false` in your `watches.yaml`.
It is recommended that you perform some type validation in Ansible on the
variables to ensure that your application is receiving expected input.

First, set a default in case the user doesn't set the `spec` field by modifying
`roles/memcached/defaults/main.yml`:
```yaml
size: 1
```

### Defining the Memcached deployment

Now that we have the spec defined, we can define what Ansible is actually
executed on resource changes. Since this is an Ansible Role, the default
behavior will be to execute the tasks in `roles/memcached/tasks/main.yml`. We
want Ansible to create a deployment if it does not exist which runs the
`memcached:1.4.36-alpine` image. Ansible 2.5+ supports the [k8s Ansible
Module](https://docs.ansible.com/ansible/2.6/modules/k8s_module.html) which we
will leverage to control the deployment definition.

Modify `roles/memcached/tasks/main.yml` to look like the following:
```yaml
---
- name: start memcached
  community.kubernetes.k8s:
    definition:
      kind: Deployment
      apiVersion: apps/v1
      metadata:
        name: '{{ meta.name }}-memcached'
        namespace: '{{ meta.namespace }}'
      spec:
        replicas: "{{size}}"
        selector:
          matchLabels:
            app: memcached
        template:
          metadata:
            labels:
              app: memcached
          spec:
            containers:
            - name: memcached
              command:
              - memcached
              - -m=64
              - -o
              - modern
              - -v
              image: "docker.io/memcached:1.4.36-alpine"
              ports:
                - containerPort: 11211

```

It is important to note that we used the `size` variable to control how many
replicas of the Memcached deployment we want. We set the default to `1`, but
any user can create a Custom Resource that overwrites the default.

### Build and run the operator

Before running the operator, Kubernetes needs to know about the new custom
resource definition the operator will be watching.

Deploy the CRD:

```sh
$ make install
```

Once this is done, there are two ways to run the operator:

- As a pod inside a Kubernetes cluster
- As a go program outside the cluster

#### 1. Run as a pod inside a Kubernetes cluster

Running as a pod inside a Kubernetes cluster is preferred for production use.

Build the memcached-operator image and push it to a registry:
```
$ make docker-build docker-push IMG=quay.io/example/memcached-operator:v0.0.1
```

Kubernetes deployment manifests are generated in `config/manager/manager.yaml`.

Deploy the memcached-operator:

```sh
$ make deploy IMG=quay.io/example/memcached-operator:v0.0.1
```

Verify that the memcached-operator is up and running:

```sh
$ kubectl get deployment -n memcached-operator-system
NAME                                        DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
memcached-operator-controller-manager       1         1         1            1           1m
```

#### 2. Run outside the cluster

This method is preferred during the development cycle to speed up deployment and testing.

**Note**: Ensure that [Ansible Runner][ansible-runner-tool] and [Ansible Runner
HTTP Plugin][ansible-runner-http-plugin] is installed or else you will see
unexpected errors from Ansible Runner when a Custom Resource is created.

Run the operator locally with the default Kubernetes config file present at
`$HOME/.kube/config`:

```sh
$ make run
```

### 3. Deploy your Operator with the Operator Lifecycle Manager (OLM)

OLM will manage creation of most if not all resources required to run your operator,
using a bit of setup from other `operator-sdk` commands. Check out the OLM integration
[user guide][quickstart-bundle] for more information.

### Create a Memcached CR

Modify `config/samples/cache_v1alpha1_memcached.yaml` as shown and create a `Memcached` custom resource:

```sh
$ cat config/samples/cache_v1alpha1_memcached.yaml
apiVersion: "cache.example.com/v1alpha1"
kind: "Memcached"
metadata:
  name: "memcached-sample"
spec:
  size: 3

$ kubectl apply -f config/samples/cache_v1alpha1_memcached.yaml
```

Ensure that the memcached-operator creates the deployment for the CR:

```sh
$ kubectl get deployment
NAME                    DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
memcached-sample        3         3         3            3           1m
```

Check the pods to confirm 3 replicas were created:

```sh
$ kubectl get pods
NAME                                  READY     STATUS    RESTARTS   AGE
memcached-sample-6fd7c98d8-7dqdr     1/1       Running   0          1m
memcached-sample-6fd7c98d8-g5k7v     1/1       Running   0          1m
memcached-sample-6fd7c98d8-m7vn7     1/1       Running   0          1m
```

### View the Ansible logs

In order to see the logs from a particular you can run:

```sh
kubectl logs deployment/memcached-operator-controller-manager
```

The logs contain the information about the Ansible run and will make it much easier to debug issues within your Ansible tasks.
Note that the logs will contain much more detailed information about the Ansible Operator's internals and interface with Kubernetes as well.

Also, you can use the environment variable `ANSIBLE_DEBUG_LOGS` set as `True` to check the full Ansible result in the logs in order to be able to debug it.

**Example**

In `config/manager/manager.yaml` and `config/default/manager_auth_proxy_patch.yaml:
```yaml
...
      containers:
      - name: manager
        env:
        - name: ANSIBLE_DEBUG_LOGS
          value: "True"
...
```

### Additional Ansible Debug

Occasionally while developing additional debug in the Operator logs is nice to have.
Using the memcached operator as an example, we can simply add the
`"ansible.sdk.operatorframework.io/verbosity"` annotation to the Custom
Resource with the desired verbosity.

```yaml
apiVersion: "cache.example.com/v1alpha1"
kind: "Memcached"
metadata:
  name: "memcached-sample"
  annotations:
    "ansible.sdk.operatorframework.io/verbosity": "4"
spec:
  size: 4
```

### Update the size

Change the `spec.size` field in the memcached CR from 3 to 4 and apply the
change:

```sh
$ cat config/samples/cache_v1alpha1_memcached.yaml
apiVersion: "cache.example.com/v1alpha1"
kind: "Memcached"
metadata:
  name: "memcached-sample"
spec:
  size: 4

$ kubectl apply -f config/samples/cache_v1alpha1_memcached.yaml
```

Confirm that the operator changes the deployment size:

```sh
$ kubectl get deployment
NAME                DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
memcached-sample    4         4         4            4           5m
```

### Cleanup

Clean up the resources:

```sh
$ make undeploy
```

**NOTE** Additional CR/CRD's can be added to the project by running the command `operator-sdk add api --api-version=cache.example.com/v1alpha1 --kind=AppService`
For more information, refer to the [`create api` CLI][create_api_cli] doc.

[ansible-install-guide]: /docs/building-operators/ansible/installation
[ansible-runner-http-plugin]:https://github.com/ansible/ansible-runner-http
[ansible-runner-tool]: https://ansible-runner.readthedocs.io/en/latest/install.html
[ansible-watches]: /docs/building-operators/ansible/reference/watches
[operator-scope]:https://v0-19-x.sdk.operatorframework.io/docs/legacy-common/operator-scope/
[layout-doc]:../reference/scaffolding
[homebrew-tool]:https://brew.sh/
[install-guide]: /docs/installation/install-operator-sdk
[git-tool]:https://git-scm.com/downloads
[go-tool]:https://golang.org/dl/
[docker-tool]:https://docs.docker.com/install/
[kubectl-tool]:https://kubernetes.io/docs/tasks/tools/install-kubectl/
<!-- TODO: update these links to sdk.operatorframework.io once 1.0.0 is released -->
[create_api_cli]: https://master.sdk.operatorframework.io/docs/cli/operator-sdk_create_api/
[quickstart-bundle]:https://master.sdk.operatorframework.io/docs/olm-integration/quickstart-bundle/
