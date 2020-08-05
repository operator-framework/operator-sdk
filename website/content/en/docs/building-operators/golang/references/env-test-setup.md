---
title: EnvTest Setup
linkTitle: EnvTest Setup
weight: 50
---

## Overview 

This document describes how to configure the environment for the [controller tests][controller-test] which uses [envtest][envtest] and is supported by the SDK. 

## Installing prerequisites

[Envtest][envtest] requires that `kubectl`, `api-server` and `etcd` be present locally. You can use this [script][script] to download these binaries into the `test/assets/` directory which will be created in your project. Update your Makefile as follows.

- Add the new `setup-envtest` makefile target:

```sh
# Setup binaries required to run the tests
setup-envtest:
        curl -sSLo setup_envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/master/hack/setup-envtest.sh
        chmod +x setup_envtest.sh
```

Then, replace your `test` target with: 

```sh
# Run tests
ENV_TEST_ASSETS_DIR=$(shell pwd)/test/assets
test: setup-envtest generate fmt vet manifests
        source setup_envtest.sh; fetch_envtest_tools $(ENV_TEST_ASSETS_DIR); setup_envtest_env $(ENV_TEST_ASSETS_DIR); go test ./... -coverprofile cover.out
```

Also, it is recommended add into the `.gitignore` a new line with `/test/assets/*` for you do not commit these binaries. 

See that you can also use your own binaries and change the location via setting up the following environment variables in your `controllers/suite_test.go`: 

```go 
var _ = BeforeSuite(func(done Done) {
	Expect(os.Setenv("TEST_ASSET_KUBE_APISERVER", "../../testbin/kube-apiserver")).To(Succeed())
	Expect(os.Setenv("TEST_ASSET_ETCD", "../../testbin/etcd")).To(Succeed())
	Expect(os.Setenv("TEST_ASSET_KUBECTL", "../../testbin/kubectl")).To(Succeed())

	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	testenv = &envtest.Environment{}

	var err error
	cfg, err = testenv.Start()
	Expect(err).NotTo(HaveOccurred())

	close(done)
}, 60)

var _ = AfterSuite(func() {
	Expect(testenv.Stop()).To(Succeed())

	Expect(os.Unsetenv("TEST_ASSET_KUBE_APISERVER")).To(Succeed())
	Expect(os.Unsetenv("TEST_ASSET_ETCD")).To(Succeed())
	Expect(os.Unsetenv("TEST_ASSET_KUBECTL")).To(Succeed())

})
```
[envtest]: https://godoc.org/sigs.k8s.io/controller-runtime/pkg/envtest
[controller-test]: https://book.kubebuilder.io/reference/writing-tests.html
[script]: https://raw.githubusercontent.com/kubernetes-sigs/kubebuilder/master/scripts/setup_envtest_bins.sh
