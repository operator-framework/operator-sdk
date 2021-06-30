---
title: Admission Webhooks
linkTitle: Webhook
weight: 30
description: An in-depth walkthough of admission webhooks.
---

## Create a validating or mutating Admission Webhook 

Admission webhooks are HTTP callbacks that receive admission requests and do something with them. It is registered with Kubernetes, and
will be called by Kubernetes to validate or mutate a resource before being stored. There are two types of admission webhooks.

#### 1. Validating admission webhook

Validating webhooks can be used to perform validations that go beyond the capabilities of OpenAPI schema validation, 
such as ensuring a field is immutable after creation or higher level permissions checks based on the user that is making 
the request to the API server. It can reject the request, but the cannot modify the object that they are receiving in the request.

#### 2. Mutating admission webhook

Mutating webhooks are most frequently used for defaulting, by adding default values for unset fields in the resource on creation. 
They can modify objects by creating a patch that will be sent back in the admission response.

For more background on Admission webhooks, refer to the [Kubebuilder documentation](https://book.kubebuilder.io/reference/admission-webhook.html) or the [official Kubernetes documentation](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/) on the topic. 
You can also refer to the [Kubebuilder webhook walkthrough](https://book.kubebuilder.io/cronjob-tutorial/webhook-implementation.html), which is similar in content to this guide. 
Kubebuilder also has a guide that walks through implementing webhooks for their example `CronJob` resource.

As an example, let's start by creating a validation webhook.
First, create an operator project and the necessary apis using `init` and `create` command of `operator-sdk`. To add a webhook to the Operator SDK project, we need to scaffold out the webhooks with the following command.

```sh
$ operator-sdk create webhook --group cache --version v1alpha1 --kind Memcached --defaulting --programmatic-validation
```

After, `create webhook` command, the following message will appear on the terminal. It scaffolds out `api/<version>/<kind>_webhook.go` file. In this example, it would be `api/v1alpha1/memcached_webhook.go`.

```sh
Writing kustomize manifests for you to edit...
Writing scaffold for you to edit...
api/v1alpha1/memcached_webhook.go
```

The `--defaulting` flag will scaffold the resources required for a mutating webhook, and the `--programmatic-validation` flag will scaffold the resources required for a validating webhook. 
In this case we scaffolded both.

After running the `create webhook` command the file structure would be:

```sh
├── Dockerfile
├── Makefile
├── PROJECT
├── api
│   └── v1alpha1
│       ├── groupversion_info.go
│       ├── memcached_types.go
│       ├── memcached_webhook.go
│       ├── webhook_suite_test.go
│       └── zz_generated.deepcopy.go
├── config
│   ├── certmanager
│   │   ├── certificate.yaml
│   │   ├── kustomization.yaml
│   │   └── kustomizeconfig.yaml
│   ├── crd
│   │   ├── kustomization.yaml
│   │   ├── kustomizeconfig.yaml
│   │   └── patches
│   │       ├── cainjection_in_memcacheds.yaml
│   │       └── webhook_in_memcacheds.yaml
│   ├── default
│   │   ├── kustomization.yaml
│   │   ├── manager_auth_proxy_patch.yaml
│   │   ├── manager_config_patch.yaml
│   │   ├── manager_webhook_patch.yaml
│   │   └── webhookcainjection_patch.yaml
│   ├── manager
│   │   ├── controller_manager_config.yaml
│   │   ├── kustomization.yaml
│   │   └── manager.yaml
│   ├── manifests
│   │   └── kustomization.yaml
│   ├── prometheus
│   │   ├── kustomization.yaml
│   │   └── monitor.yaml
│   ├── rbac
│   │   ├── auth_proxy_client_clusterrole.yaml
│   │   ├── auth_proxy_role.yaml
│   │   ├── auth_proxy_role_binding.yaml
│   │   ├── auth_proxy_service.yaml
│   │   ├── kustomization.yaml
│   │   ├── leader_election_role.yaml
│   │   ├── leader_election_role_binding.yaml
│   │   ├── memcached_editor_role.yaml
│   │   ├── memcached_viewer_role.yaml
│   │   ├── role_binding.yaml
│   │   └── service_account.yaml
│   ├── samples
│   │   ├── cache_v1alpha1_memcached.yaml
│   │   └── kustomization.yaml
│   ├── scorecard
│   │   ├── bases
│   │   │   └── config.yaml
│   │   ├── kustomization.yaml
│   │   └── patches
│   │       ├── basic.config.yaml
│   │       └── olm.config.yaml
│   └── webhook
│       ├── kustomization.yaml
│       ├── kustomizeconfig.yaml
│       └── service.yaml
├── controllers
│   ├── memcached_controller.go
│   └── suite_test.go
├── go.mod
├── go.sum
├── hack
│   └── boilerplate.go.txt
└── main.go

19 directories, 53 files
```

The scaffolded file `api/v1alpha1/memcached_webhook.go` has method signatures which need to be implemented for the validation webhook.

The example memcached operator explained in the [tutorial](https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/) is too simple to require defaulting or additional validations,
but as an example, we can reinforce that the default value for `spec.size` should be `3`, by adding the following logic to 
the `Default` function (note that this is already handled by the CRD defaulting and is technically completely superfluous):

```sh
// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Memcached) Default() {
	log.Info("default", "name", r.Name)

	if r.Spec.Size == 0 {
		r.Spec.Size = 3
	}
}
```

For validation, we can enforce that the size of the cluster follows a rule that is difficult or impossible 
to describe with OpenAPI, for example, that the size of the cluster always remain an odd number.

In order to perform this validation, we can simply add the `validateOdd` function shown below to  perform the check.

```sh
func validateOdd(n int32) error {
	if n%2 == 0 {
		return errors.New("Cluster size must be an odd number")
	}
	return nil
}
```

At the end, let's call this method from `ValidateCreate` and `ValidateUpdate` function as shown below.

```sh
func (r *Memcached) ValidateCreate() error {
	log.Info("validate create", "name", r.Name)
	return validateOdd(r.Spec.Size)
}

func (r *Memcached) ValidateUpdate(old runtime.Object) error {
	log.Info("validate update", "name", r.Name)
	return validateOdd(r.Spec.Size)
}
```

This function gets called whenever an object deletion happens.

```sh
func (r *Memcached) ValidateDelete() error {
	log.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
```

### Generate webhook manifests and enable webhook deployment 

Once your webhooks are implemented, all that’s left is to create the WebhookConfiguration manifests required to register your webhooks with Kubernetes:

```sh
$ make manifests
```

You will need to enable cert-manager and webhook deployment in order to deploy these webhooks properly. 
To do so, edit the `config/default/kustomize.yaml` and uncomment the sections marked by `[WEBHOOK]` and `[CERTMANAGER]` comments. 
More detail on this step can be found in the [Kubebuilder documentation](https://book.kubebuilder.io/cronjob-tutorial/running-webhook.html).

### Update main.go so that running locally works 

Refer upstream [Kubebuilder doc](https://book.kubebuilder.io/cronjob-tutorial/main-revisited.html?highlight=enable_webhooks#you-said-something-about-main) for more details.

## Run your operator and webhooks 

#### Run locally

Technically, The webhooks can be run locally, but for it to work you need to generate certificates for the webhook server and store them at `/tmp/k8s-webhook-server/serving-certs/tls.{crt,key}`. Generally it’s easier to just disable them locally and test the webhooks when running in a cluster.

If your certificates are properly configured, you should be able to start your operator by running:

```sh
$ make run ENABLE_WEBHOOKS=true
```

For more details about running webhook locally, refer [here](https://book.kubebuilder.io/cronjob-tutorial/running.html#running-webhooks-locally).

#### Run as a Deployment inside the cluster

For instructions on deploying your operator into a cluster, refer to the [tutorial](https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/#2-run-as-a-deployment-inside-the-cluster) instructions. Adding webhooks does not alter this step.

## Create a Memcached CR to exercise your webhook

First, follow the instructions for creating your Memcached CR in the [tutorial](https://sdk.operatorframework.io/docs/building-operators/golang/tutorial).

Once you have completed this step, you should have a Memcached CR with a size of 3 and a Memcached deployment with 3 replicas in your cluster.

To ensure that you are in the correct state, run the following commands and verify that the output roughly matches:

```
$ kubectl get memcached/memcached-sample -o yaml
apiVersion: cache.example.com/v1alpha1
kind: Memcached
metadata:
  clusterName: ""
  creationTimestamp: 2018-03-31T22:51:08Z
  generation: 0
  name: memcached-sample
  namespace: default
  resourceVersion: "245453"
  selfLink: /apis/cache.example.com/v1alpha1/namespaces/default/memcacheds/memcached-sample
  uid: 0026cc97-3536-11e8-bd83-0800274106a1
spec:
  size: 3
status:
  nodes:
  - memcached-sample-6fd7c98d8-7dqdr
  - memcached-sample-6fd7c98d8-g5k7v
  - memcached-sample-6fd7c98d8-m7vn7
```

```
$ kubectl get deployment
NAME                                    READY   UP-TO-DATE   AVAILABLE   AGE
memcached-operator-controller-manager   1/1     1            1           8m
memcached-sample                        3/3     3            3           3m
```

#### Update the size

Update `config/samples/cache_v1alpha1_memcached.yaml` to change the `spec.size` field in the Memcached CR from `3` to `5`:

```sh
$ kubectl patch memcached memcached-sample -p '{"spec":{"size": 5}}' --type=merge
```

Confirm that the operator changes the deployment size:

```
$ kubectl get deployment
NAME                                    READY   UP-TO-DATE   AVAILABLE   AGE
memcached-operator-controller-manager   1/1     1            1           10m
memcached-sample                        5/5     5            5           3m
```

#### Update the size to an even number

Update `config/samples/cache_v1alpha1_memcached.yaml` to change the `spec.size` field in the Memcached CR from `5` to `4`:

```sh
$ kubectl patch memcached memcached-sample -p '{"spec":{"size": 4}}' --type=merge
```

The request should fail, and you should see a response like this:

```sh
Error from server (Cluster size must be an odd number): admission webhook "vmemcached.kb.io" denied the request: Cluster size must be an odd number
```

This means that your update was rejected by your webhook. If you inspect the cluster state, you will see that your resources are unchanged:

```
$ kubectl get deployment
NAME                                    READY   UP-TO-DATE   AVAILABLE   AGE
memcached-operator-controller-manager   1/1     1            1           10m
memcached-sample                        5/5     5            5           3m
```