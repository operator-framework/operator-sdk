---
title: Writing Kuttl Scorecard Tests
weight: 50
---

# Kuttl Tests using Operator SDK Scorecard

This guide outlines the steps which can be followed to implement scorecard
tests using the [kuttl][kuttl] project.

## Run scorecard with kuttl tests:

### kuttl test image

The kuttl project binary is include in the scorecard-test-kuttl test image
which is released as part of the operator-sdk release.  You can find it
at quay.io/operator-framework/scorecard-test-kuttl

### kuttl test directory structure

The kuttl program when run looks for tests that have the following
structure:

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

```

1. `bundle/` - Contains bundle manifests and metadata under test.
2. `bundle/tests/scorecard/config.yaml` - Configuration yaml to define and run scorecard tests.
3. `tests/kuttl` - Contains tests written for kuttl to execute
4. `tests/kuttl/kuttl-test.yaml` - Contains the kuttl configuration 
5. `tests/kuttl/list-pods` - Contains a kuttl test suite

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


### kuttl Tests

The kuttl test tool looks for tests within a test suite directory
to follow a naming convention as follows:
```
        └── kuttl
            ├── kuttl-test.yaml
            └── list-pods
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

### Building the project

The project makefile is to help us build the go project and test image using docker. An example of the [makefile][sample_makefile] script can be found in the sample test image.

To build the project, use the `docker build` command and specify the desired name of the image in the format: `<repository_name>/<username>/<image_name>:tag`.

Push the image to a remote repository by running the docker command:

```
docker push <repository_name>/<username>/<image_name>:tag
```

### Running scorecard command

The `operator-sdk alpha scorecard` command is used to execute the scorecard tests by specifying the location of test bundle in the command. The name or suite of the tests which are to be executed can be specified with the `--selector` flag. The command will create scorecard pods with the image specified in `config.yaml` for the respective test. For example, the `CustomTest1Name` test provides the following json output.

```console
operator-sdk alpha scorecard bundle/ --selector=suite=custom -o json --wait-time=32s --skip-cleanup=false
{
  "metadata": {
    "creationTimestamp": null
  },
  "log": "",
  "results": [
    {
      "name": "customtest1",
      "description": "",
      "state": "pass",
      "log": "an ISV custom test"
    }
  ]
}
```

**Note**: More details on the usage of `operator-sdk alpha scorecard` command and its flags can be found in the [scorecard user documentation][user_doc]

### Debugging scorecard custom tests

The `--skip-cleanup` flag can be used when executing the `operator-sdk alpha scorecard` command to cause the scorecard created test pods to be unremoved. 
This is useful when debugging or writing new tests so that you can view
the test logs or the pod manifests.

### Scorecard initContainer

The scorecard inserts an `initContainer` into the test pods it creates. The
`initContainer` serves the purpose of uncompressing the operator bundle
contents, mounting them into a shared mount point accessible by test
images.  The operator bundle contents are stored within a ConfigMap, uniquely
built for each scorecard test execution.  Upon scorecard completion,
the ConfigMap is removed as part of normal cleanup, along with the test
pods created by scorecard.

### Using Custom Service Accounts

Scorecard does not deploy service accounts, RBAC resources, or
namespaces for your test but instead considers these resources
to be outside its scope.  You can however implement whatever
service accounts your tests require and then specify
that service account from the command line using the service-account flag:
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

### Returning Multiple Test Results

Some custom tests might require or be better implemented to return
more than a single test result.  For this case, scorecard's output
API allows multiple test results to be defined for a single test.  See
https://github.com/operator-framework/operator-sdk/blob/master/pkg/apis/scorecard/v1alpha3/types.go#L59

### Accessing the Kube API

Within your custom tests you might require connecting to the Kube API.  
In golang, you could use the [client-go][client_go] API for example to 
check Kube resources within your tests, or even create custom resources. Your 
custom test image is being executed within a Pod, so you can use an in-cluster 
connection to invoke the Kube API.


[client_go]: https://github.com/kubernetes/client-go
[kuttl]: https://kuttl.dev
[kuttl_yaml]: https://kuttl.dev/docs/cli.html#examples
[kuttl_tests]: https://kuttl.dev/docs/kuttl-test-harness.html#writing-your-first-test
