---
title: Reconcile Period from Custom Resource Annotations
linkTitle: Custom Resource Annotations Reconcile Period
weight: 200
description: Allow a user to set the desired reconcile period from the custom resource's annotations
---

While running a Helm-based operator, the reconcile-period can be specified through the custom resource's annotations under the `helm.sdk.operatorframework.io/reconcile-period` key. 
This feature guarantees that an operator will get reconciled, at minimum, in the specified interval of time. In other words, it ensures that the cluster will not go longer
than the specified reconcile-period without being reconciled. However, the cluster may be reconciled at any moment if there are changes detected in the desired state.

The reconcile period can be specified in the custom resource's annotations in the following manner: 

```sh
...
metadata:
  name: nginx-sample
  annotations:
    helm.sdk.operatorframework.io/reconcile-period: 5s
...
```

The value that is present under this key must be in the h/m/s format. For example, 1h2m4s, 3m0s, 4s are all valid values, but 1x3m9s is invalid. 

**NOTE**: This is just one way of specifying the reconcile period for Helm-based operators. There are two other ways: using the `--reconcile-period` command-line flag and under the 'reconcilePeriod'
key in the watches.yaml file. If these three methods are used simultaneously to specify reconcile period (which they should not be), the order of precedence is as follows. Custom Resource Annotations > watches.yaml > command-line flag.
