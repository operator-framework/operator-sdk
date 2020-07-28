---
title: Custom Resource Annotations in Helm-based Operators
linkTitle: Custom Resource Annotations
weight: 300
description: Use custom resource annotations to configure how reconciliation works.
---

## `helm.sdk.operatorframework.io/upgrade-force`

This annotation can be set to `"true"` on custom resources to enable the chart to be upgraded with the
`helm upgrade --force` option. For more info see the [Helm Upgrade documentation](https://helm.sh/docs/helm/helm_upgrade/)
and this [explanation](https://github.com/helm/helm/issues/7082#issuecomment-559558318) of `--force` behavior.

**Example**

```yaml
apiVersion: example.com/v1alpha1
kind: Nginx
metadata:
  name: nginx-sample
  annotations:
    helm.sdk.operatorframework.io/upgrade-force: "true"
spec:
  replicaCount: 2
  service:
    port: 8080
```

Setting this annotation to `true` and making a change to trigger an upgrade (e.g. setting `spec.replicaCount: 3`)
will cause the custom resource to be reconciled and upgraded with the `force` option. This can be verified in the
log message when an upgrade succeeds:

```
{"level":"info","ts":1591198931.1703992,"logger":"helm.controller","msg":"Upgraded release","namespace":"helm-nginx","name":"example-nginx","apiVersion":"cache.example.com/v1alpha1","kind":"Nginx","release":"example-nginx","force":true}
```
