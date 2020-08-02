---
title: EnvTest Setup
linkTitle: EnvTest Setup
weight: 50
---

## Overview 

This document describes how to configure the environment for the [controller tests][controller-test] which uses [envtest][envtest] and is supported by the SDK. 

## Installing prerequisites

[Envtest][envtest] requires that `kubectl`, `api-server` and `etcd` binaries be present locally. To download and configure these binaries, run the following:

```sh
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed 's/x86_64/amd64/')
curl -fsL "https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-1.16.4-${OS}-${ARCH}.tar.gz" -o kubebuilder-tools
tar -zvxf kubebuilder-tools
sudo mv kubebuilder/ /usr/local/kubebuilder
```

See that you can also use your own binaries and change the location via setting up the the following environment variables in your `controllers/suite_test.go`: 

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
