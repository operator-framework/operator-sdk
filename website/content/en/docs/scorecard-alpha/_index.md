---
title: Scorecard (alpha)
weight: 25
---

# operator-sdk scorecard (alpha)

## Overview

The scorecard command, part of the operator-sdk, executes tests 
on your operator based upon a user defined configuration file 
and test images.

Tests are implemented within test images that are configured 
and constructed to be executed by scorecard.

Each test(s) is executed within a Pod by scorecard, pod logs 
contain the test results, and scorecard aggregates the test 
results to display to end users.  Scorecard therefore assumes 
it is being executed with access to a configured Kube cluster.

Scorecard has built-in basic and OLM tests but also provides a 
means to execute custom user defined tests.

## Requirements

The scorecard tests make no assumptions as to the state of the 
operator being tested.  Creating operators and custom resources 
for an operator are left outside the scope of the scorecard itself.

Scorecard tests can however create whatever resources they 
require if the tests are designed for resource creation.

Scorecard requires the following Kube environment:

- Access to a Kubernetes v1.17.1+ cluster

## Running the Scorecard

1. Define a scorecard configuration file `config.yaml`.  A sample 
configuration file [config.yaml][sample-config] is found within 
the SDK repository. See [Config file](#config-file) for an explaination 
of the configuration file format.  Unless you are executing custom 
tests, you can just copy the provided example configuration file 
into your project.
2. Place the scorecard configuration file within your project 
bundle directory a the following location `tests/scorecard/config.yaml`.  
You can override the default location of the configuration file 
by specifying the `--config` flag.
3. Execute the [`scorecard` command][cli-scorecard]. See the 
[Command args](#command-args) to check its options.

## Configuration

The scorecard test execution is driven by a configuration file, 
`config.yaml`.   The configuration file is located at the following 
location within your bundle:
```
tests/scorecard/config.yaml
```

### Config File

A complete scorecard configuration file can be found [here][sample-config] 
and used for running the scorecard pre-defined tests that ship with the SDK.

A sample of the scorecard configuration file may look as follows:

```yaml
tests:
- name: "basic-check-spec"
  image: quay.io/operator-framework/scorecard-test:dev
  entrypoint:
  - scorecard-test
  - basic-check-spec
  labels:
    suite: basic
    test: basic-check-spec-test
  description: check the spec test
- name: "olm-bundle-validation"
  image: quay.io/operator-framework/scorecard-test:dev
  entrypoint:
  - scorecard-test
  - olm-bundle-validation
  labels:
    suite: olm
    test: olm-bundle-validation-test
  description: validate the bundle test
```

The configuration file defines each test that scorecard can execute.  
The following fields of the scorecard configuration file define the 
test as follows:
 
 * name - the human readable name you use to describe the test, 
however, the actual name printed by scorecard when it executes the 
test is defined within the test itself
 * image - the test container image name that implements a test 
 * entrypoint - the command and arguments that are invoked in the 
test image to execute a test
 * labels - user defined labels that allow for test selection using 
the scorecard CLI `--selector` flag
 * description - the human readable description for the test, however, 
the description printed by scorecard on an executed test is the 
description provided by the test implementation itself

### Command Args

The scorecard command has the following syntax:
```
operator-sdk alpha scorecard [bundle path] | [bundle image name] [flags]
```

The scorecard requires a positional argument that holds either the
on-disk path to your operator bundle or the name of a bundle image.

Scorecard flags include the following:

| Flag        | Type   | Description   |
| --------    | -------- | -------- |
| `--config`  | string | path to config file (default `<bundle directory>/tests/scorecard/config.yaml`; file type and extension must be `.yaml`). If a config file is not provided and a config file is not found at the default location, the scorecard will exit with an error. |
| `--kubeconfig`, `-o`  | string |  path to kubeconfig used by the scorecard to execute tests, if not supplied, the KUBECONFIG environment variable or $HOME/.kube/config path is assumed, if those are not found an in-cluster configuraiton is assumed. |
| `--list`, `-L`  | bool |  if true, only print the test names that would be run based on selector filtering. |
| `--namespace`, `-n`  | string |  the namespace for scorecard to use when executing test Pods |
| `--output`, `-o`  | string | output format. Valid options are: `text` and `json`. The default format is `text`, which is designed to be a simpler human readable format.|
| `--selector`, `-l`  | string |  the label selector to filter tests on. |
| `--service-account`, `-s`  | string |  the service account for scorecard to use when creating test Pods.|
| `--skip-cleanup`, `-x`  | boolean |  disable deleting test Pods and ConfigMaps generated from executing scorecard tests, useful for debugging.|
| `--wait-time`, `-w`  | int |  seconds to wait for tests to complete.  Example is 35s.|

## Tests Performed

Scorecard users can specify a `--selector` CLI flag to filter which 
tests it will execute.  If a selector flag is not supplied, then all 
the tests within the scorecard configuration file are executed.

Tests are executed serially, one after the other, with test results 
being aggregated by scorecard to present back to the end user.

Here are some examples of supplying a selector flag to scorecard:
```sh
operator-sdk scorecard -o text --selector=test=basic-check-spec-test
operator-sdk scorecard -o text --selector=suite=olm
operator-sdk scorecard -o text --selector='test in (basic-check-spec-test,olm-bundle-validation-test)'
```

### Basic Operator

| Test        | Description   | Test Name |
| --------    | -------- | -------- |
| Spec Block Exists | This test checks the Custom Resource(s) created in the cluster to make sure that all CRs have a spec block. | basic-check-spec-test |

### OLM Integration

| Test        | Description   | Short Name |
| --------    | -------- | -------- |
| OLM Bundle Validation | This test validates the OLM bundle manifests found in the bundle that is passed into scorecard.  If the bundle contents contain errors, then the test result output will include the validator log as well as error messages from the validation library.  See this [document][olm-bundle] for details on OLM bundles.| olm-bundle-validation-test |
| Provided APIs have validation |This test verifies that the CRDs for the provided CRs contain a validation section and that there is validation for each spec and status field detected in the CR. | olm-crds-have-validation |
| Owned CRDs Have Resources Listed | This test makes sure that the CRDs for each CR provided via the `cr-manifest` option have a `resources` subsection in the [`owned` CRDs section][owned-crds] of the CSV. If the test detects used resources that are not listed in the resources section, it will list them in the suggestions at the end of the test. | olm-crds-have-resources |
| Spec Fields With Descriptors | This test verifies that every field in the Custom Resources' spec sections have a corresponding descriptor listed in the CSV.| olm-spec-descriptors-test |
| Status Fields With Descriptors | This test verifies that every field in the Custom Resources' status sections have a corresponding descriptor listed in the CSV.| olm-status-descriptors-test |

## Scorecard Output

### JSON format

See an example of the JSON format produced by a scorecard test:

```json
{
  "metadata": {
    "creationTimestamp": null
  },
  "log": "",
  "results": [
    {
      "name": "basic-check-spec",
      "description": "Custom Resource has a Spec Block",
      "state": "pass"
    }
  ]
}
```

### Text format

See an example of the text format produced by a scorecard test:

```
	basic-check-spec                    : pass
	CR: 
	Labels: 
```

This output is structured according to the golang struct found
[here][scorecard-struct].

## Exit Status

The scorecard return code is 1 if any of the tests executed did not 
pass and 0 if all selected tests pass.

## Extending the Scorecard with Custom Tests

Scorecard can execute custom tests provided by ISVs or end users if 
the custom tests following some mandated conventions including:

 * tests are implemented within a container image
 * tests accept an entrypoint which include a command and arguments
 * tests produce v1alpha2 scorecard output in JSON format with no extraneous logging in the test output
 * tests can obtain the bundle contents at a shared mount point of /bundle
 * tests can access the Kube API using an in-cluster client connection

See here for an example of a custom test image written in golang.  
Writing custom tests in other programming languages is possible 
if the test image follows the above guidelines.



[sample-config]: https://github.com/operator-framework/operator-sdk/blob/master/internal/scorecard/alpha/testdata/bundle/tests/scorecard/config.yaml
[scorecard-struct]: https://github.com/operator-framework/operator-sdk/blob/master/pkg/apis/scorecard/v1alpha2/types.go
[olm-bundle]:https://github.com/operator-framework/operator-registry#manifest-format
