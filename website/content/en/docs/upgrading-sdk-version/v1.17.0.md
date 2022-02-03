---
title: v1.17.0
weight: 998983000
---

## (go/v3) Upgrade go and dependencies

Upgrade controller-runtime from `0.10.0` to `0.11.0`, k8s from `1.22` to `1.23` and controller-gen from `v0.0.7` to `v0.8.0`.
Following the steps to update your project.
1) Update the go.mod file with:

```
go 1.17

require (
  github.com/onsi/ginkgo v1.16.5
  github.com/onsi/gomega v1.17.0
  k8s.io/api v0.23.0
  k8s.io/apimachinery v0.23.0
  k8s.io/client-go v0.23.0
  sigs.k8s.io/controller-runtime v0.11.0
)
```
2) Run `go mod tidy`

Now, let's update the Makefile targets accordingly:

1) Replace `ENVTEST_K8S_VERSION = 1.22` with `ENVTEST_K8S_VERSION = 1.23`
2) Replace `sigs.k8s.io/controller-tools/cmd/controller-gen@v0.7.0` with `sigs.k8s.io/controller-tools/cmd/controller-gen@v0.8.0`
3) Run `make manifests` and `make generate` to ensure that you will update your manifests with the new versions
4) You might want to run `make all` to ensure that all updates were done accordinly and succeffully

_See [#5505](https://github.com/operator-framework/operator-sdk/pull/5505) for more details._

## Reduce debug log level for the sidecar container kube-rbac-proxy from 10 to 0

Update the file `config/default/manager_auth_proxy_patch.yaml` by replacing `"--v=10"` with `"--v=0"`

_See [#5505](https://github.com/operator-framework/operator-sdk/pull/5505) for more details._

## Add resource requests and limits to kube-rbac-proxy

Update the file `config/default/manager_auth_proxy_patch.yaml` by adding:

```
...
resources:
  limits:
    cpu: 500m
    memory: 128Mi
  requests:
    cpu: 5m
    memory: 64Mi
```

_See [#5505](https://github.com/operator-framework/operator-sdk/pull/5505) for more details._
