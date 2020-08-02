---
title: Admission Webhooks
linkTitle: Admission Webhooks
weight: 40
---

## Create a validating or mutating Admission Webhook

An admission webhook is an HTTP callback that is registered with Kubernetes, and will be called by Kubernetes to validate or mutate
a resource before being stored. There are two types of admission webhooks, validating and mutating. Validating webhooks can be used
to perform validations that go beyond the capabilities of OpenAPI schema validation, such as ensuring a field is immutable after
creation or higher level permissions checks based on the user that is making the request to the API server. Mutating webhooks are
most frequently used for defaulting, by adding default values for unset fields in the resource on creation.

For more background on Admission webhooks, refer to the [Kubebuilder documentation][kubebuilder_admission_controllers] 
or the [official Kubernetes documentation][kubernetes_admission_controllers] on the topic. You can also refer to the
[Kubebuilder webhook walkthrough][kubebuilder_cronjob_webhook], which is similar in content to this guide.
Kubebuilder also has a guide
that walks through implementing webhooks for their example `CronJob` resource.

To add a webhook to your Operator SDK project, first you must scaffold out the webhooks with the following command.

```sh
$ operator-sdk create webhook --group cache --version v1alpha1 --kind Memcached --defaulting --programmatic-validation
```

The `--defaulting` flag will scaffold  the resources required for a mutating webhook, and the `--programmatic-validation` flag will
scaffold the resources required for a validating webhook. In this case we scaffolded both.

To implement the actual webhook logic, edit the `api/v1alpha1/memcached_webhook.go` file. The file will
contain some boilerplate to set up the logger and register your webhook with the controller manager, as
well as a variety of unimplemented methods (marked with `TODO`s). The mutating webhook implementation
belongs in the `Default` function. The validating webhook implementation will be split between the
`ValidateCreate`, `ValidateUpdate`, and `ValidateDelete` functions, which allows you to perform different
validations based on the operation being performed, ie, preventing a field from being changed on `Update`. 

The memcached operator we are building in this example is too simple to require defaulting or
additional validations, but as an example, we can reinforce that the default value for `spec.size` should be `3`, by adding the following logic to the `Default` function (note that this is already handled by the CRD defaulting and is technically completely superfluous):

```go
if r.Spec.Size == 0 {
    r.Spec.Size = 3
}
```

For validation, we can enforce that the size of the cluster follows a rule that is difficult or impossible
to describe with OpenAPI, for example, that the size of the cluster always remain an odd number.

To do so, we can simply implement a new function in `api/v1alpha1/memcached_webhook.go`, which performs this check:

```go
func validateOdd(n int32) error {
	if n%2 == 0 {
		return errors.New("Cluster size must be an odd number")
	}
	return nil
}
```

And then, call that method from the `ValidateCreate` and `ValidateUpdate` functions:

```go
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

### Generate webhook manifests and enable webhook deployment

Once your webhooks are implemented, all that's left is to create the `WebhookConfiguration` manifests
required to register your webhooks with Kubernetes:

```sh
$ make manifests
```

You will need to enable cert-manager and webhook deployment in order to deploy these webhooks properly.
To do so, edit the `config/default/kustomize.yaml` and uncomment the sections marked by `[WEBHOOK]` and `[CERTMANAGER]` comments. More detail on this step can be found in the 
[Kubebuilder documentation][kubebuilder_running_webhook].

### Update main.go so that running locally works

To ensure that running locally continues working, ensure that there is a check that prevents
the webhooks from being started when the `ENABLE_WEBHOOKS` flag is set to false. To do so,
edit `main.go`, and add the following check around the call to `SetupWebhookWithManager` if it's not already present:

```go
if os.Getenv("ENABLE_WEBHOOKS") != "false" {
	if err = (&cachev1alpha1.Memcached{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Memcached")
		os.Exit(1)
	}
}
```

## Run your operator and webhooks

### Run locally

Technically, The webhooks can be run locally, but for it to work you need to generate certificates for the webhook server
and store them at `/tmp/k8s-webhook-server/serving-certs/tls.{crt,key}`. Generally it's easier to just disable
them locally and test the webhooks when running in a cluster.

If your certificates are properly configured, you should be able to start your operator by running:

```sh
$ make run ENABLE_WEBHOOKS=true
```

### Run as a Deployment inside the cluster

For instructions on deploying your operator into a cluster, refer to the [tutorial][tutorial_run_as_deployment] instructions.
Adding webhooks does not alter this step.


## Create a Memcached CR to exercise your webhook

First, follow the instructions for creating your Memcached CR in the [tutorial][tutorial_create_a_cr].

Once you have completed this step, you should have a Memcached CR with a size of 3 and a Memcached deployment
with 3 replicas in your cluster.

To ensure that you are in the correct state, run the following commands and verify that the output
roughly matches:

```console
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

```console
$ kubectl get deployment
NAME                                    READY   UP-TO-DATE   AVAILABLE   AGE
memcached-operator-controller-manager   1/1     1            1           8m
memcached-sample                        3/3     3            3           1m
```

### Update the size

Update `config/samples/cache_v1alpha1_memcached.yaml` to change the `spec.size` field in the Memcached CR from 3 to 5:

```sh
$ kubectl patch memcached memcached-sample -p '{"spec":{"size": 5}}' --type=merge
```

Confirm that the operator changes the deployment size:

```console
$ kubectl get deployment
NAME                                    READY   UP-TO-DATE   AVAILABLE   AGE
memcached-operator-controller-manager   1/1     1            1           10m
memcached-sample                        5/5     5            5           3m
```

#### Update the size to an even number

Update `config/samples/cache_v1alpha1_memcached.yaml` to change the `spec.size` field in the Memcached CR from 3 to 4:

```sh
$ kubectl patch memcached memcached-sample -p '{"spec":{"size": 4}}' --type=merge
```

The request should fail, and you should see a response like this:

```sh
Error from server (Cluster size must be an odd number): admission webhook "vmemcached.kb.io" denied the request: Cluster size must be an odd number
```

This means that your update was rejected by your webhook. If you inspect the cluster state, you will see that your
resources are unchanged:

```console
$ kubectl get deployment
NAME                                    READY   UP-TO-DATE   AVAILABLE   AGE
memcached-operator-controller-manager   1/1     1            1           10m
memcached-sample                        5/5     5            5           3m
```

[tutorial_run_as_deployment]: /docs/building-operators/golang/tutorial/#2-run-as-a-deployment-inside-the-cluster
[tutoria_create_a_cr]: /docs/building-operators/golang/tutorial/#create-a-memcached-cr

[kubebuilder_admission_controllers]: https://book.kubebuilder.io/reference/admission-webhook.html
[kubebuilder_cronjob_webhook]: https://book.kubebuilder.io/cronjob-tutorial/webhook-implementation.html
[kubebuilder_running_webhook]: https://book.kubebuilder.io/cronjob-tutorial/running-webhook.html
[kubernetes_admission_controllers]: https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/
