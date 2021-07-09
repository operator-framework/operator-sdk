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
the request to the API server. It can reject the request, but it cannot modify the object that they are receiving in the request.

#### 2. Mutating admission webhook

Mutating webhooks are most frequently used for defaulting, by adding default values for unset fields in the resource on creation. 
They can modify objects by creating a patch that will be sent back in the admission response.

For more background on Admission webhooks, refer to the [Kubebuilder documentation](https://book.kubebuilder.io/reference/admission-webhook.html) or the [official Kubernetes documentation](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/) on the topic. 
You can also refer to the [Kubebuilder webhook walkthrough](https://book.kubebuilder.io/cronjob-tutorial/webhook-implementation.html), which is similar in content to this guide. 
Kubebuilder also has a guide that walks through implementing webhooks for their example `CronJob` resource.

### Create Validation Webhook

As an example, let's start by creating a validation webhook.

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
│       ├── memcached_webhook.go
│       ├── webhook_suite_test.go
├── config
│   ├── certmanager
│   │   ├── certificate.yaml
│   │   ├── kustomization.yaml
│   │   └── kustomizeconfig.yaml
│   ├── default
│   │   ├── manager_webhook_patch.yaml
│   │   └── webhookcainjection_patch.yaml
│   └── webhook
│       ├── kustomization.yaml
│       ├── kustomizeconfig.yaml
│       └── service.yaml
├── go.mod
├── go.sum
└── main.go
```

The scaffolded file `api/v1alpha1/memcached_webhook.go` has method signatures which need to be implemented for the validation webhook.

Following this, there are a few steps which need to be done in your operator project to enable webhhoks. This will involve:

1. Implementing the required methods for Validating or Mutating webhook in <kind>_webhook.go. An example of such implementation is provided [here](https://book.kubebuilder.io/cronjob-tutorial/webhook-implementation.html).

2. Uncommenting sections in config/default/kustomization.yaml to enable webhook and cert-manager configuration through kustomize. Cert-manager (or any third party solution) can be used to provision certificates for webhook server. This is explained in detail here.

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

There are two ways to test your operator project with webhooks.

#### Run locally

Technically, the webhooks can be run locally, but for it to work you need to generate certificates for the webhook server and store them at /tmp/k8s-webhook-server/serving-certs/tls.{crt,key}. For more details about running webhook locally, refer [here](https://book.kubebuilder.io/cronjob-tutorial/running.html#running-webhooks-locally).

#### Run as a Deployment inside the cluster

Adding webhooks does not alter deploying your operator. For instructions on deploying your operator into a cluster, refer to the [tutorial](https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/#2-run-as-a-deployment-inside-the-cluster).

## Exercise your webhook

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