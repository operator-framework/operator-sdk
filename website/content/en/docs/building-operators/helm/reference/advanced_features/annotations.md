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

## `helm.sdk.operatorframework.io/uninstall-wait`

This annotation can be set to `"true"` on custom resources to enable the deletion to wait until all the resources in the
`status.deployedRelease.manifest` are deleted. 

**Example**

```yaml
apiVersion: example.com/v1alpha1
kind: Nginx
metadata:
  name: nginx-sample
  annotations:
    helm.sdk.operatorframework.io/uninstall-wait: "true"
spec:
...
status:
  ...
  deployedRelease:
    manifest: |
      ---
      # Source: nginx/templates/serviceaccount.yaml
      apiVersion: v1
      kind: ServiceAccount
      metadata:
        name: nginx-sample
        labels:
          helm.sh/chart: nginx-0.1.0
          app.kubernetes.io/name: nginx
          app.kubernetes.io/instance: nginx-sample
          app.kubernetes.io/version: "1.16.0"
          app.kubernetes.io/managed-by: Helm
      ---
      # Source: nginx/templates/service.yaml
      apiVersion: v1
      kind: Service
      metadata:
        name: nginx-sample
        labels:
          helm.sh/chart: nginx-0.1.0
          app.kubernetes.io/name: nginx
          app.kubernetes.io/instance: nginx-sample
          app.kubernetes.io/version: "1.16.0"
          app.kubernetes.io/managed-by: Helm
      spec:
        type: ClusterIP
        ports:
          - port: 80
            targetPort: http
            protocol: TCP
            name: http
        selector:
          app.kubernetes.io/name: nginx
          app.kubernetes.io/instance: nginx-sample
      ---
      # Source: nginx/templates/deployment.yaml
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: nginx-sample
        labels:
          helm.sh/chart: nginx-0.1.0
          app.kubernetes.io/name: nginx
          app.kubernetes.io/instance: nginx-sample
          app.kubernetes.io/version: "1.16.0"
          app.kubernetes.io/managed-by: Helm
      spec:
        replicas: 1
        selector:
          matchLabels:
            app.kubernetes.io/name: nginx
            app.kubernetes.io/instance: nginx-sample
        template:
          metadata:
            labels:
              app.kubernetes.io/name: nginx
              app.kubernetes.io/instance: nginx-sample
          spec:
            serviceAccountName: nginx-sample
            securityContext:
              {}
            containers:
              - name: nginx
                securityContext:
                  {}
                image: "nginx:1.16.0"
                imagePullPolicy: IfNotPresent
                ports:
                  - name: http
                    containerPort: 80
                    protocol: TCP
                livenessProbe:
                  httpGet:
                    path: /
                    port: http
                readinessProbe:
                  httpGet:
                    path: /
                    port: http
                resources:
                  {}
```

Setting this annotation to `true` and deleting the custom resource will cause the custom resource to be reconciled
continuously until all the resources in `status.deployedRelease.manifest` are deleted. This can be verified in the
log message when a delete has been triggered:

```
{"level":"info","ts":1612294054.5845876,"logger":"helm.controller","msg":"Uninstall wait","namespace":"default","name":"nginx-sample","apiVersion":"example.com/v1alpha1","kind":"Nginx","release":"nginx-sample"}

```
