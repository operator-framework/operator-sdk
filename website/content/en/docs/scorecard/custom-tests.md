---
title: Writing Custom Scorecard Tests
weight: 50
---

# Custom Tests using Operator SDK Scorecard

This guide outlines the steps which can be followed to extend the existing scorecard tests and implement operator specific custom tests. 

## Run scorecard with custom tests:

### Building test image:

The following steps explain creating of a custom test image which can be used with Scorecard to run operator specific tests. As an example, let us start by creating a sample go repository containing the test bundle data, custom scorecard tests and a Makefile to help us build a test image.

The sample test image repository present [here][custom_scorecard_repo] has the following project structure:

```
├── Makefile
├── bundle
│   ├── manifests
│   │   ├── cache.example.com_memcached_crd.yaml
│   │   └── memcached-operator.clusterserviceversion.yaml
│   ├── metadata
│   │   └── annotations.yaml
│   └── tests
│       └── scorecard
│           └── config.yaml
├── go.mod
├── go.sum
├── images
│   └── custom-scorecard-tests
│       ├── Dockerfile
│       ├── bin
│       │   ├── entrypoint
│       │   └── user_setup
│       ├── cmd
│       │   └── test
│       │       └── main.go
│       └── custom-scorecard-tests
└── internal
    └── tests
        └── tests.go
```

1. `bundle/` - Contains bundle manifests and metadata under test.
2. `bundle/tests/scorecard/config.yaml` - Configuration yaml to define and run scorecard tests.
3. `images/custom-scorecard-tests/cmd/test/main.go` - Scorecard test binary.
4. `internal/tests/tests.go` -  Contains the implementation of custom tests specific to the operator.

#### Writing custom test logic:

Scorecard currently implements a few [basic][basic_tests] and [olm][olm_tests] tests for the image bundle, custom resources and custom resource definitions. Additional tests specific to the operator can also be included in the test suite of scorecard.

The `tests.go` file is where the custom tests are implemented in the sample test image project. These tests use `scapiv1alpha2.ScorecardTestResult` struct to populate the result, which is then converted to json format for the output. For example, the format of a simple custom sample test can be as follows:

```Go
package tests

import (
	"github.com/operator-framework/operator-registry/pkg/registry"
	scapiv1alpha2 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"
)

const (
	CustomTest1Name = "customtest1"
)

// CustomTest1 
func CustomTest1(bundle registry.Bundle) scapiv1alpha2.ScorecardTestResult {
	r := scapiv1alpha2.ScorecardTestResult{}
	r.Name = CustomTest1Name
	r.Description = "Custom Test 1"
	r.State = scapiv1alpha2.PassState
	r.Errors = make([]string, 0)
	r.Suggestions = make([]string, 0)

  // Implement relevant custom test logic here

	return r
}
```

### Scorecard Configuration file:

The [configuration file][config_yaml] includes test definitions and metadata to run the test. For the example `CustomTest1` function, the following fields should be specified in `config.yaml`.

```yaml
stages:
- tests:
  - image: quay.io/username/custom-scorecard-tests:dev
    entrypoint:
    - custom-scorecard-tests
    - customtest1
    labels:
      suite: custom
      test: customtest1
 ```

The important fields to note here are:
1. `image` - name and tag of the test image which was specified in the Makefile.
2. `labels` - the name of the `test` and `suite` the test function belongs to. This can be specified in the `operator-sdk alpha scorecard` command to run the desired test.

**Note**: The default location of `config.yaml` inside the bundle is `<bundle directory>/tests/scorecard/config.yaml`. It can be overridden using the `--config` flag. For more details regarding the configuration file refer to [user docs][user_doc].

### Scorecard binary:

The scorecard test image implementation requires the bundle under test to be present in the test image. The `GetBundle()` function reads the pod's bundle to fetch the manifests and scorecard configuration from desired path.

```Go
	cfg, err := GetBundle("/bundle")
	if err != nil {
		log.Fatal(err.Error())
	}
```
The scorecard binary uses `config.yaml` file to locate tests and execute the them as Pods which scorecard creates. Custom test images are included into Pods that scorecard creates, passing in the bundle contents on a shared mount point to the test image container. The specific custom test that is executed is driven by the config.yaml's entry-point command and arguments.

An example scorecard binary is present [here][scorecard_binary].

The names with which the tests are identified in `config.yaml` and would be passed in the `scorecard` command, are to be specified here.

```Go
...     
switch entrypoint[0] {
case tests.CustomTest1Name:
    result = tests.CustomTest1(*cfg)
    ...
}
...
```

The result of the custom tests which is in `scapiv1alpha2.ScorecardTestResult` format, is converted to json for output.

```Go
prettyJSON, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		log.Fatal("Failed to generate json", err)
	}
fmt.Printf("%s\n", string(prettyJSON))
```

The names of the custom tests are also included in `printValidTests()` function:

```Go
func printValidTests() (result v1alpha2.ScorecardTestResult) {
...
    str := fmt.Sprintf("Valid tests for this image include: %s,
            tests.CustomTest1Name
    result.Errors = append(result.Errors, str")
...

```

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
[olm_tests]: https://github.com/operator-framework/operator-sdk/blob/master/internal/scorecard/alpha/tests/olm.go
[basic_tests]: https://github.com/operator-framework/operator-sdk/blob/master/internal/scorecard/alpha/tests/basic.go
[config_yaml]: https://github.com/operator-framework/operator-sdk/blob/master/internal/scorecard/alpha/testdata/bundle/tests/scorecard/config.yaml
[scorecard_main_func]: https://github.com/operator-framework/operator-sdk/blob/master/images/scorecard-test/cmd/test/main.go
[custom_scorecard_repo]: https://github.com/operator-framework/operator-sdk/tree/master/internal/scorecard/alpha/examples
[user_doc]: /docs/scorecard/scorecard-alpha/
[scorecard_binary]: https://github.com/operator-framework/operator-sdk/blob/master/internal/scorecard/alpha/examples/custom-scorecard-tests/images/custom-scorecard-tests/cmd/test/main.go
[sample_makefile]: https://github.com/operator-framework/operator-sdk/blob/master/internal/scorecard/alpha/examples/custom-scorecard-tests/Makefile
