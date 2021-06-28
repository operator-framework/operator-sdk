---
title: Admission Webhooks
linkTitle: Webhook
weight: 30
description: An in-depth walkthough of admission webhooks.
---

## Create a validating or mutating Admission Webhook 

Admission webhooks are HTTP callbacks that receive admission requests and do something with them. It is registered with Kubernetes, and
will be called by Kubernetes to validate or mutate a resource before being stored. There are two types of admission webhooks.

#### 1. Validation admission webhook

Validating webhooks can be used to perform validations that go beyond the capabilities of OpenAPI schema validation, 
such as ensuring a field is immutable after creation or higher level permissions checks based on the user that is making 
the request to the API server. It can reject the request, but the cannot modify the object that they are receiving in the request.

#### 2. Mutating admission webhook

Mutating webhooks are most frequently used for defaulting, by adding default values for unset fields in the resource on creation. 
They can modify objects by creating a patch that will be sent back in the admission response.

For more background on Admission webhooks, refer to the [Kubebuilder documentation](https://book.kubebuilder.io/reference/admission-webhook.html) or the [official Kubernetes documentation](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/) on the topic. 
You can also refer to the [Kubebuilder webhook walkthrough](https://book.kubebuilder.io/cronjob-tutorial/webhook-implementation.html), which is similar in content to this guide. 
Kubebuilder also has a guide that walks through implementing webhooks for their example `CronJob` resource.

First, creat an operator using `init` and `create` command of `operator-sdk`. To add a webhook to your Operator SDK project, first you must scaffold out the webhooks with the following command.

```sh
$ operator-sdk create webhook --group cache --version v1alpha1 --kind Memcached --defaulting --programmatic-validation
```

After, `create webhook` command below message will appear on the terminal. It scaffolds out `api/v1alpha1/memcached_webhook.go` file.

```sh
Writing kustomize manifests for you to edit...
Writing scaffold for you to edit...
api/v1alpha1/memcached_webhook.go
```

The `--defaulting` flag will scaffold the resources required for a mutating webhook, and the `--programmatic-validation` flag will scaffold the resources required for a validating webhook. 
In this case we scaffolded both.

After running the `create webhook` command the file structure will change to match the one shown as below.

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
├── bin
│   └── controller-gen
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

The scaffoled file `api/v1alpha1/memcached_webhook.go` has `ValidateCreate`, `ValidateUpdate`, and `ValidateDelete` functions, 
which allows you to perform different validations based on the operation being performed. The mutating webhook implementation belongs in the `Default` function.

The memcached operator we are building in this example is too simple to require defaulting or additional validations, 
but as an example, we can reinforce that the default value for `spec.size` should be `3`, by adding the following logic to 
the `Default` function (note that this is already handled by the CRD defaulting and is technically completely superfluous):

```sh
// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Memcached) Default() {
	memcachedlog.Info("default", "name", r.Name)

	if r.Spec.Size == 0 {
		r.Spec.Size = 3
	}
}
```

For validation, we can enforce that the size of the cluster follows a rule that is difficult or impossible 
to describe with OpenAPI, for example, that the size of the cluster always remain an odd number.

In order to perform this validation, we can simply add below `validateOdd` function that will perform this check.

```sh
func validateOdd(n int32) error {
	if n%2 == 0 {
		return errors.New("Cluster size must be an odd number")
	}
	return nil
}
```

At the end, call this method from `ValidateCreate` and `ValidateUpdate` function as shown below.

```sh
// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Memcached) ValidateCreate() error {
	memcachedlog.Info("validate create", "name", r.Name)
	return validateOdd(r.Spec.Size)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Memcached) ValidateUpdate(old runtime.Object) error {
	memcachedlog.Info("validate update", "name", r.Name)
	return validateOdd(r.Spec.Size)
}
```

This function gets called whenever there is an object deletion happens.

```sh
// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Memcached) ValidateDelete() error {
	memcachedlog.Info("validate delete", "name", r.Name)

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

To ensure that operator running locally continues working, ensure that there is a check that prevents the webhooks from being started when the `ENABLE_WEBHOOKS` flag is set to `false`. To do so, edit `main.go`, and add the following check around the call to `SetupWebhookWithManager` if it’s not already present:

```sh
if os.Getenv("ENABLE_WEBHOOKS") != "false" {
	if err = (&cachev1alpha1.Memcached{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Memcached")
		os.Exit(1)
	}
}
```

## Run your operator and webhooks 

#### Run locally

Technically, The webhooks can be run locally, but for it to work you need to generate certificates for the webhook server and store them at `/tmp/k8s-webhook-server/serving-certs/tls.{crt,key}`. Generally it’s easier to just disable them locally and test the webhooks when running in a cluster.

If your certificates are properly configured, you should be able to start your operator by running:

```sh
$ make run ENABLE_WEBHOOKS=true
```

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
memcached-sample   
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
memcached-sample   
```

#### Update the size to an even number

Update `config/samples/cache_v1alpha1_memcached.yaml` to change the `spec.size` field in the Memcached CR from `3` to `4`:

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
memcached-sample
```