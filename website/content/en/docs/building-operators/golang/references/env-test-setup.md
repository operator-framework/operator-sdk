---
title: EnvTest Setup
linkTitle: EnvTest Setup
weight: 50
---

## Overview 

This document describes how to configure the environment for the [controller tests][controller-test] supported by the SDK.

## Configuring your test 

The suite test requires specify the `kubectl`, `api-server` and `etcd` the k8s binaries which are used by it. You can use the following script as a helper which will create the `testbin/` directory with these binaries. Following an example by adding a new target in your Makefile. 

```sh
# Setup binaries required to run the tests
# The script will create the testbin dir and add on it the required binaries
# See that it expects the Kubernetes and ETCD version
testsetup:
	curl -sSLo setup_envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/kubebuilder/master/scripts/setup_envtest_bins.sh 
	chmod +x setup_envtest.sh
	./setup_envtest.sh v1.18.2 v3.4.3
```


Also, note that the above script will use environment variables to specify the place where its binaries can be found:

```shell
  export TEST_ASSET_KUBECTL=<kubectl bin path>
  export TEST_ASSET_KUBE_APISERVER=<api-server bin path>
  export TEST_ASSET_ETCD=/<etd bin path>
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
 
[controller-test]: https://book.kubebuilder.io/reference/writing-tests.html