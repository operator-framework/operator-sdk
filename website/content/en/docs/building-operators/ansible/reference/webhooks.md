---
title: Adding Admission Webhooks to an Ansible-based Operator
linkTitle: Webhooks
weight: 20
---

For general background on what admission webhooks are, why to use them, and how to build them,
please refer to the official Kubernetes documentation on [Extensible Admission Controllers][admission-controllers]

This guide will assume that you understand the above content, and that you have an existing admission
webhook server. You will likely need to make a few modifications to the webhook server container.

When integrating an admission webhook server into your Ansible-based Operator, we recommend that you
deploy it as a sidecar container alongside your operator. This allows you to make use of the proxy
server that the operator deploys, as well as the cache that backs it.

## Ensuring the webhook server uses the caching proxy

When an Ansible-based Operator runs, it creates a Kubernetes proxy server and serves it on
`http://localhost:8888`. This proxy server does not require any authorization, so all you need to
do to make use of the proxy is ensure that your Kubernetes client is pointing at `http://localhost:8888`
and that it does not attempt to verify SSL. If you use the default in-cluster configuration, you will
be hitting the real API server and will not get caching for free.

## Deploying the webhook server

Create a new file called `config/default/manager_webhook_patch.yaml` with the following content
(making sure to replace the image reference placeholder string):

```yaml
# This patch injects a sidecar container which is an admission webhook for the
# controller manager.
apiVersion: apps/v1
kind: Deployment
metadata:
  name: controller-manager
  namespace: system
spec:
  template:
    spec:
      containers:
      - name: webhook
        # Replace this with the built image name
        image: "REPLACE_WEBHOOK_IMAGE"
        volumeMounts:
        - mountPath: /etc/tls/
          name: webhook-cert
      volumes:
      # This assumes there is a secret called webhook-cert containing TLS certificates
      # Projects like cert-manager can create these certificates
      - name: webhook-cert
        secret:
          secretName: webhook-cert
```

Then, update `config/default/kustomization.yaml` to include this patch:

```yaml
patchesStrategicMerge:
- manager_webhook_patch.yaml # Add this line
```

Now, when deploying the operator with `make deploy`, your webhook server will run alongside the
operator, but Kubernetes will not yet call the webhooks before resources can be created. In order
to let Kubernetes know about your webhooks, you must create specific API resources.


<!--
   TODO(fabianvf,asmacdo) update these sections to direct the user
     to create files in the config directory and make use of kustomize.
     The Go plugin's webhook scaffolding might be a good reference.
-->
## Making Kubernetes call your webhooks

In order to make your webhooks callable at all, first you must create a `Service` that points at your
webhook server. Below is a sample service that creates a `Service` named `my-operator-webhook`, that will
send traffic on port `443` to port `5000` in a `Pod` that matches the selector `name=my-operator`. Modify these
values to match your environment.

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-operator-webhook
spec:
  ports:
  - name: webhook
    port: 443
    protocol: TCP
    # Change targetPort to match the port your server is listening on
    targetPort: 5000
  selector:
    # Change this selector to match the labels on your operator pod
    name: my-operator
  type: ClusterIP
```

Now that you have a `Service` directing traffic to your webhook server, you will need to create
[`MutatingWebhookConfiguration`][mutating-webhook] or [`ValidatingWebhookConfiguration`][validating-webhook] objects (depending on what type of webhook you have deployed), which will tell Kubernetes
to send certain API requests through your webhooks before writing to etcd.

Below are examples of both [`MutatingWebhookConfiguration`][mutating-webhook] and [`ValidatingWebhookConfiguration`][validating-webhook] objects,
which will tell Kubernetes to call the `my-operator-webhook` service when `samples.example.com Example` resources
are created. The mutating webhook is served on the `/mutating` path in my example webhook server, and the validating webhook is served on `/validating`. Update these values as needed to reflect your environment
and desired behavior. These objects are thoroughly documented in the official Kubernetes documentation on [Extensible Admission Controllers][admission-controllers]

```yaml
---
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: mutating.example.com
webhooks:
- name: "mutating.example.com"
  rules:
  - apiGroups:   ["samples.example.com"]
    apiVersions: ["*"]
    operations:  ["CREATE"]
    resources:   ["examples"]
    scope:       "Namespaced"
  clientConfig:
    service:
      # Replace this with the namespace your service is in
      namespace: REPLACE_NAMESPACE
      name: my-operator-webhook
      path: /mutating
  admissionReviewVersions: ["v1"]
  sideEffects: None
---
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: validating.example.com
webhooks:
- name: validating.example.com
  rules:
  - apiGroups:   ["samples.example.com"]
    apiVersions: ["*"]
    operations:  ["CREATE"]
    resources:   ["examples"]
    scope:       "Namespaced"
  clientConfig:
    service:
      # Replace this with the namespace your service is in
      namespace: REPLACE_NAMESPACE
      name: my-operator-webhook
      path: /validating
  admissionReviewVersions: ["v1"]
  failurePolicy: Fail
  sideEffects: None
```

If these resources are configured properly you will now have an admissions webhook that can reject or mutate
incoming resources before they are written to the Kubernetes database.

## Summary

To deploy an existing admissions webhook to validate or mutate your Kubernetes resources alongside an
Ansible-based Operator, you must
1. Configure your admissions webhook to use the proxy server running on `http://localhost:8888` in the operator pod
1. Add the webhook container to your operator deployment
1. Create a `Service` pointing to your webhook
1. Make sure your webhook is reachable via the `Service` over `https`
1. Create [`MutatingWebhookConfiguration`][mutating-webhook] or [`ValidatingWebhookConfiguration`][validating-webhook] mapping the resource you want to mutate/validate to the `Service` you created


[admission-controllers]:https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/
[validating-webhook]:https://v1-26.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#validatingwebhookconfiguration-v1-admissionregistration-k8s-io
[mutating-webhook]:https://v1-26.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#mutatingwebhookconfiguration-v1-admissionregistration-k8s-io
