---
title: Envtest Setup
linkTitle: Envtest Setup
description: Learn how to setup your project to run integration tests using envtest
weight: 50
---

By default, Go-based operators are scaffolded to make use of controller-runtime's [`envtest`][envtest] framework, which uses `kubectl`, `kube-apiserver`, and `etcd` to simulate the API portions of a real cluster. You can use [this script][script] to download these binaries into the `testbin/` directory and configure your environment to use them. Update your `Makefile` by replacing your `test` target with: 

```sh
# Run tests
ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: generate fmt vet manifests
        mkdir -p ${ENVTEST_ASSETS_DIR}
        test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/master/hack/setup-envtest.sh
        source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out
```

If using `git`, it is recommended to add `testbin/*` to your `.gitignore` file to avoid committing these binaries. 

[envtest]: https://godoc.org/sigs.k8s.io/controller-runtime/pkg/envtest
[controller-test]: https://book.kubebuilder.io/reference/writing-tests.html
[script]: https://raw.githubusercontent.com/kubernetes-sigs/kubebuilder/master/scripts/setup_envtest_bins.sh
