---
title: v1.9.0
weight: 998991000
---

## The ansible-operator binary will reject the kubeconfig if the server URL contains a path.

There is currently no workaround other than running the operator as a pod in the cluster (where it will use the internal endpoint). The fix for this issue is currently blocked waiting on a fix to the apimachinery package. Once the upstream issue is merged then a proper fix will be done for ansible-operator. Work on this issue is being tracked here: https://github.com/operator-framework/operator-sdk/issues/4925

_See [#4987](https://github.com/operator-framework/operator-sdk/pull/4987) for more details._
