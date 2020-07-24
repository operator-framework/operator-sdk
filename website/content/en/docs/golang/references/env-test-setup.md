---
title: EnvTest Setup
linkTitle: EnvTest Setup
weight: 50
---

## Overview 

This document describes how to configure the environment for the [controller tests][controller-test] which uses [envtest][envtest] and is supported by the SDK. 

## Installing prerequisites

[Envtest][envtest] requires that `kubectl`, `api-server` and `etcd` be present locally. You can use this [script][script] to download these binaries into the `testbin/` directory. It may be convenient to add this script your Makefile as follows:

```sh
# Setup binaries required to run the tests
# See that it expects the Kubernetes and ETCD version
K8S_VERSION = v1.18.2
ETCD_VERSION = v3.4.3
testbin:
	curl -sSLo setup_envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/kubebuilder/master/scripts/setup_envtest_bins.sh 
	chmod +x setup_envtest.sh
	./setup_envtest.sh $(K8S_VERSION) $(ETCD_VERSION)
```


The above script sets these environment variables to specify where test binaries can be found. In case you would like to not use the script then, is possible to do the same configuration to inform the path of your binaries: 

```shell
$ export TEST_ASSET_KUBECTL=<kubectl-bin-path>
$ export TEST_ASSET_KUBE_APISERVER=<api-server-bin-path>
$ export TEST_ASSET_ETCD=<etcd-bin-path>
``` 

See that the environment variables also can be specified via your `controllers/suite_test.go` such as the following example. 

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