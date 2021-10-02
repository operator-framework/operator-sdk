---
title: Writing Kuttl Scorecard Tests
weight: 50
---

This guide outlines the steps which can be followed to implement scorecard
tests using the [kuttl][kuttl] project and specifically the scorecard
kuttl test image.

## Defining kuttl Tests in Scorecard

Scorecard users can include kuttl tests within their operator
bundles as follows:
```
$ tree ./bundle
./bundle
├── manifests
│   ├── cache.example.com_memcacheds_crd.yaml
│   └── memcached-operator.clusterserviceversion.yaml
├── metadata
│   └── annotations.yaml
└── tests
    └── scorecard
        ├── config.yaml
        └── kuttl
            ├── kuttl-test.yaml
            └── list-pods
                ├── 00-assert.yaml
                └── 00-pod.yaml
            └── list-other
                ├── 00-assert.yaml
                └── 00-pod.yaml
```

 * `bundle/` - Contains bundle manifests and metadata under test.
 * `bundle/tests/scorecard/config.yaml` - Configuration yaml to define and run scorecard tests.
 * `bundle/tests/scorecard/kuttl` - Contains tests written for kuttl to execute
 * `bundle/tests/scorecard/kuttl/kuttl-test.yaml` - Contains the kuttl configuration, it is here that you would add any kuttl specific configuration settings that you might require.
 * `bundle/tests/scorecard/kuttl/list-pods` - Contains a kuttl test case
 * `bundle/tests/scorecard/kuttl/list-pods/00-assert.yaml` - Contains a kuttl test case assert
 * `bundle/tests/scorecard/kuttl/list-pods/00-pod.yaml` - Contains a kuttl test case step
 * `bundle/tests/scorecard/kuttl/list-other` - Contains another kuttl test case

When the scorecard kuttl binary is executed, it will process all the test
cases under the scorecard/kuttl directory within the bundle contents.

## Configuring kuttl Tests in the Scorecard Configuration

In the scorecard configuration file, you might have the following
definition of what the selector `suite=kuttlsuite` will translate to:
```yaml
stages:
- tests:
  - image: quay.io/operator-framework/scorecard-test-kuttl:v2.0.0
    labels:
      suite: kuttlsuite
      test: kuttltest1
```

This test configuration will execute the scorecard-test-kuttl
image which executes kuttl.  The kuttl output is translated
into scorecard compliant output which is displayed back to the
end user along with any other test results.

With the above kuttl test configuration, you can execute that
kuttl test using scorecard as follows:
```bash
operator-sdk scorecard <bundle_dir_or_image> --selector=suite=kuttlsuite
```

## Defining kuttl Specific Configuration Options

The [kuttl configuration file][kuttl_yaml] is documented within the
kuttl project.

An example of the kuttl configuration file is as follows:

```yaml
apiVersion: kudo.dev/v1beta1
kind: TestSuite
parallel: 4
timeout: 120
startControlPlane: false
```

The important fields to note here are:

 * `startControlPlane` - Set to false since scorecard assumes it is running
within a control plane already.

Other kuttl configurations settings are available for more advanced kuttl
use cases.  See [kuttl configuration][kuttl_configuration] for more details on kuttl configuration.

### kuttl Tests Explained

The kuttl test tool looks for tests to execute within the bundle
following a naming convention as follows:
```
        └── kuttl
            ├── kuttl-test.yaml
            └── list-pods
                ├── 00-assert.yaml
                └── 00-pod.yaml
            └── list-other
                ├── 00-assert.yaml
                └── 00-pod.yaml
```

The important fields to note here are:

 * `kuttl-test.yaml` - The name required for your kuttl configuration file.
 * `list-pods, list-other` - The names given by you for these test cases.
 * `00-assert.yaml` - The assert file is executed to test whether or
not the test was successful, this assertion determines whether or not
the test passed or failed.
 * `00-pod.yaml` - The pod file is used to define what the test will
create, in this case a pod will be created based on the manifest within
00-pod.yaml.

The number in front of the assert and pod manifests is used to determine
the order in which kuttl will execute the files.

See [kuttl tests][kuttl_tests] for a detailed description of how
kuttl tests are named and executed.

### kuttl Test Privileges

The kuttl tests a user might write can vary widely in functionality
and in particular require special Kubernetes RBAC privileges outside
of what the default service account for a namespace might have.
It is therefore very likely you will be required to run scorecard
in a custom service account that holds the required RBAC permissions,
like `config/rbac/service_account.yaml` in Go operator projects.
You can specify a custom service account in scorecard as follows:

```console
$ operator-sdk scorecard <bundle_dir_or_image> --service-account=my-project-controller-manager
```

Also, you can specify a non-default namespace that scorecard will run in:

```console
$ operator-sdk scorecard <bundle_dir_or_image> --namespace=my-project-system
```

If you do not specify either of these flags, the default namespace
and service account will be used by the scorecard to run test pods.

It is worth noting that scorecard-test-kuttl specifies a namespace
to the kubectl-kuttl command which causes kuttl to not create a
namespace for each test.  This might impact your kuttl tests in
that you might need to perform resource cleanup in your tests
instead of depending upon namespace deletion to perform that cleanup.

Also of note is that in our example [kuttl configuration][kuttl_configuration]
file, we add the `suppressLog: events` setting which means that
kuttl will not log kubernetes events and thereby means you do not
have to provide RBAC access for reading kubernetes events to the
service account used to run kuttl tests.

[client_go]: https://github.com/kubernetes/client-go
[kuttl]: https://kuttl.dev
[kuttl_yaml]: https://kuttl.dev/docs/cli.html#examples
[kuttl_tests]: https://kuttl.dev/docs/kuttl-test-harness.html#writing-your-first-test
[kuttl_configuration]:https://kuttl.dev/docs/testing/reference.html#testsuite
