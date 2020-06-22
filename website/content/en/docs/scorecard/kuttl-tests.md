---
title: Writing Kuttl Scorecard Tests
weight: 50
---

# Kuttl Tests using Operator SDK Scorecard

This guide outlines the steps which can be followed to implement scorecard
tests using the [kuttl][kuttl] project and specifically the scorecard
kuttl test image.

## Run scorecard with kuttl tests

### kuttl test image

The kuttl project provides a Docker container which contains the 
kubectl-kuttl binary.  That binary is included in the scorecard-test-kuttl 
test image which is released as part of the operator-sdk release.  
The scorecard kuttl test image is found at quay.io/operator-framework/scorecard-test-kuttl.

### kuttl test directory structure within a bundle

The kuttl program when run by scorecard looks for kuttl tests in the
bundle as follows:
```
.
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

1. `bundle/` - Contains bundle manifests and metadata under test.
2. `bundle/tests/scorecard/config.yaml` - Configuration yaml to define and run scorecard tests.
3. `tests/kuttl` - Contains tests written for kuttl to execute
4. `tests/kuttl/kuttl-test.yaml` - Contains the kuttl configuration, it is here that you would add any kuttl specific configuration settings that you might require.
5. `tests/kuttl/list-pods` - Contains a kuttl test case
6. `tests/kuttl/list-other` - Contains another kuttl test case

When the kuttl binary is executed, it will process all the test
cases under the scorecard/kuttl directory within the bundle contents.

### kuttl Configuration file:

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
1. `startControlPlane` - Set to false since scorecard assumes it is running
within a control plane already.

Other kuttl configurations settings are available for more advanced kuttl
use cases.  See [kuttl configuration][kuttl_configuration] for more details on kuttl configuration.

### kuttl Tests

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
1. `00-assert.yaml` - The assert file is executed to test whether or
not the test was successful, this assertion determines whether or not
the test passed or failed.  
2. `00-pod.yaml` - The pod file is used to define what the test will
create, in this case a pod will be created based on the manifest within
00-pod.yaml.

The number in front of the assert and pod manifests is used to determine
the order in which kuttl will execute the files.

See [kuttl tests][kuttl_tests] for a detailed description of how
kuttl tests are named and executed.

### kuttl test output

When kuttl runs, it produces output such as the following:
```json
{
   "name": "",
   "tests": 2,
   "failures": 1,
   "time": "32.638117203s",
   "testsuite": [
     {
       "tests": 2,
       "failures": 1,
       "time": "31.523609155s",
       "name": "/bundle/tests/scorecard/kuttl",
       "testcase": [
         {
           "classname": "kuttl",
           "name": "list-pods",
           "time": "1.147508681s",
           "assertions": 1
         },
         {
           "classname": "kuttl",
           "name": "list-pods2",
           "time": "31.521237551s",
           "assertions": 1,
           "failure": {
             "text": "resource Pod:kudo-test-hot-chamois/: .metadata.labels.app: value mismatch, expected: nginy != actual: nginx",
             "message": "failed in step 0-pod"
           }
         }
       ]
     }
   ]
 }
```

This output in JSON format is processed by scorecard and converted into
the required scorecard output format.

### How kuttl tests are executed

The scorecard will run kuttl tests if you specify the kuttl
test image in your scorecard configuration file and also
specify the kuttl test(s) to be run.  For example, you 
could enter the following scorecard command:
```bash
operator-sdk alpha scorecard deploy/olm-catalog/memcached-operator --selector=suite=kuttlsuite 
```

This command causes tests that match `suite=kuttlsuite` to be executed.  In
the scorecard configuration file, you might have the following
definition of what `suite=kuttlsuite` will translate to:
```yaml
tests:
- name: "kuttltest1"
  image: quay.io/operator-framework/scorecard-test-kuttl:dev
  labels:
    suite: kuttlsuite
    test: kuttltest1
  description: an ISV custom test that does...
```

Within the scorecard-test-kuttl image, the following kuttl command
is executed:
```bash
kubectl-kuttl test /bundle/tests/scorecard/kuttl/ --namespace=$SCORECARD_NAMESPACE --report=JSON --artifacts-dir=/tmp
```

This command references the bundle contents of your operator, and
runs the kuttl tests on test definitions found under `/bundle/tests/scorecard/kuttl`, and reports the result in JSON format at `/tmp/kuttl-report.json`.

The json report will then be read by the scorecard-test-kuttl binary
which will format the kuttl test results into the expected scorecard
test output format.

Notice that we are not having kuttl create namespaces when it runs
tests, instead we are having it execute within the namespace that
scorecard is running within.

### kuttl test privileges

The kuttl tests can vary widely in functionality and in particular
require special Kubernetes RBAC priviledges outside of what your
service account might have.  So, you will want to take note of
the service account you are going to be running scorecard and its kuttl
tests with to see if you might require additional privileges.  For
example, if your kuttl test requires the ability to  create namespaces, then
you will likely need to create a custom service account to run
scorecard with that includes the required RBAC permissions.


You can specify a custom service account in scorecard as follows:
```
operator-sdk alpha scorecard --service-account=mycustomsa
```

Also, you can set up a non-default namespace that your tests
will be executed within using the following namespace flag:
```
operator-sdk alpha scorecard --namespace=mycustomns
```

If you do not specify either of these flags, the default namespace
and service account will be used by the scorecard to run test pods.

### Accessing the Kube API

The kuttl utility wnen ran, will execute against the Kube API using
an in-cluster Kube connection.


[client_go]: https://github.com/kubernetes/client-go
[kuttl]: https://kuttl.dev
[kuttl_yaml]: https://kuttl.dev/docs/cli.html#examples
[kuttl_tests]: https://kuttl.dev/docs/kuttl-test-harness.html#writing-your-first-test
[kuttl_configuration]:https://kuttl.dev/docs/testing/reference.html#testsuite
