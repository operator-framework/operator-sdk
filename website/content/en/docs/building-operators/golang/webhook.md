---
title: Admission Webhooks
linkTitle: Webhook
weight: 30
description: An in-depth walkthrough of admission webhooks.
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

For more background on Admission webhooks, refer to the [Kubebuilder documentation](https://book.kubebuilder.io/reference/admission-webhook.html) or the [official Kubernetes documentation](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/). 
You can also refer to the [Kubebuilder webhook walkthrough](https://book.kubebuilder.io/cronjob-tutorial/webhook-implementation.html), which is similar in content to this guide. 

### Create Validation Webhook

As an example, let's walk through the scaffolding of a validation webhook for the sample memcached operator.

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
In this case we have scaffolded both.

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

Following this, there are a few steps which need to be done in your operator project to enable webhooks. This will involve:

1. Implementing the required methods for Validating or Mutating webhook in `<kind>_webhook.go`. An example of such implementation is provided [here](https://book.kubebuilder.io/cronjob-tutorial/webhook-implementation.html).

2. Uncommenting sections in `config/default/kustomization.yaml` to enable webhook and cert-manager configuration through kustomize. Cert-manager (or any third party solution) can be used to provision certificates for webhook server. This is explained in detail [here](https://book.kubebuilder.io/cronjob-tutorial/running-webhook.html#deploy-webhooks).

**Note**
If OLM is being used to deploy the operator, then the section prefixed with `[CERT-MANAGER]` need not be uncommented. This is because, OLM currently handles the cert generation and rotation for webhook deployment using self-signed certs. It also does not allow users to specify the name or mount location for the certs. More documentation on this issue can be found [here](https://olm.operatorframework.io/docs/advanced-tasks/adding-admission-and-conversion-webhooks/#deploying-an-operator-with-webhooks-using-olm).

### Generate webhook manifests and enable webhook deployment 

Once your webhooks are implemented, all that’s left is to create the `WebhookConfiguration` manifests required to register your webhooks with Kubernetes:

```sh
$ make manifests
```

## Run your operator and webhooks 

There are two ways to test your operator project with webhooks.

#### Run locally

Technically, the webhooks can be run locally, but for it to work you need to generate certificates for the webhook server and store them at `/tmp/k8s-webhook-server/serving-certs/tls.{crt,key}`. For more details about running webhook locally, refer [here](https://book.kubebuilder.io/cronjob-tutorial/running.html#running-webhooks-locally).

#### Run as a Deployment inside the cluster

Adding webhooks does not alter deploying your operator. For instructions on deploying your operator into a cluster, refer to the [tutorial](https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/#2-run-as-a-deployment-inside-the-cluster).
