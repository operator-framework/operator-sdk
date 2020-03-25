---
title: Scorecard Custom Test Framework
authors:
  - "@jemccorm"
reviewers:
  - TBD
  - "@joelandford"
  - "@zeus"
approvers:
  - TBD
  - "@shurley"
  - "@dmesser"
creation-date: 2020-03-19
last-updated: 2020-03-19
status: implementable
---

# Scorecard - Custom Tests


## Release Signoff Checklist

- \[ \] Enhancement is `implementable`
- \[ \] Design details are appropriately documented from clear requirements
- \[ \] Test plan is defined
- \[ \] Graduation criteria for dev preview, tech preview, GA
- \[ \] User-facing documentation is created 

## Summary

This enhancement proposal outlines the SDK scorecard’s design for running of custom tests, specifically tests that are outside of what Red Hat includes by default within the scorecard’s tests.   The proposed design would provide both a means to write custom tests and also convert the existing scorecard tests (basic and olm) into the custom test format.

## Motivation

The scorecard needs a robust means of enabling end-users of the SDK to write and execute their own operator tests.   Ideally, the custom test framework would decouple test logic from within the SDK itself, making the SDK and scorecard logic in particular more flexible and maintainable in the long run.

## Goals

 * Document a design for scorecard to execute end-user or custom tests.

 * Document stories/steps required to deliver scorecard custom test functionality.

### Non-Goals

 * This is a design proposal and does not include a proof-of-concept.

## Proposal

This design proposal is to establish a pattern for implementing custom scorecard tests and also support the goal of migrating existing scorecard tests into the same format as the custom tests.

With this proposal, the scorecard will continue to setup and teardown the Operator being tested, as well as creating Custom Resources being tested.

### User Stories

#### Story 1

I am as an Operator developer, I'd like to be able to construct custom tests and execute them via the scorecard.

#### Story 2

Scorecard will ship with an example custom test that end users could use as a reference when writing their custom tests.

#### Story 3

As a scorecard developer I would like for custom tests and internal scorecard tests to share the same architecture to provide a cleaner scorecard design and implementation. Scorecard internal tests will migrate to the custom test format and therefore be externalized from the scorecard binary itself.

#### Story 4

I am as an Operator developer, would like to be able to use the Scorecard feature to test the [OLM Bundles](https://github.com/operator-framework/operator-registry/tree/v1.5.3#manifest-format) created for my project.  This story changes Scorcard to require a bundle image name be passed as a configuration CLI flag.  Scorecard will look for example CRs within the bundle's CSV instead of the scorecard configuration.  Scorecard will look for CRDs within the bundle's CSV instead of the scorecard configuration as part of this story.

#### Story 5

I am as an Operator developer, will require the ability to include scorecard confgration file and scorecard test pod manifests into an operator bundle image. 

#### Story 6 

Scorecard would support creating and deleting namespaces to run tests within.  When a test is executed, a unique namespace will be created to run the test within.  When the test is completed, the namespace would be removed by scorecard.  So, if you have 5 tests to be run, scorecard would create a unique namespace for each of those 5 tests.

#### Story 7 (optional)

- I am as an Operator developer, would like to be able to create tests using the same Assert syntax adopted by [Kuttl](https://github.com/kudobuilder/kuttl) and check its results aggregated within scorecard results. 

**Implementation Notes**
The Scorecard would support the creation of a Kuttl test image and the execute tests via Kuttl, however, it would be transparent for users since its results will be integrated into scorecard results.

## Design Details

### Custom Test Definition

* A custom test is any sort of test logic required, written in any programming language that satifies the scorecard test requirements for output format and execution.

### Custom Test Components

This proposal outlines the end-user facing components that make up a custom test including:

#### Test Image

A custom test is based on a container image, that image is created by
the end-user using whatever tools they want to use.

The test image must produce container log output that conforms to the scorecard Test Result output.  The output is an array of Test Results, so a test image could contain multiple tests if necessary.  Here is a sample of test output in the required format:
```
    [
      {
	      "name": "Spec fields with descriptors",
	      "description": "All spec fields have matching descriptors in the CSV",
	      "state": "fail",
	      "suggestions": [
		"Add a spec descriptor for size"
	      ],
	      "crname": "example-memcached"
      }, 
      {
	      "name": "Some other test",
	      "description": "Another test on the CSV",
	      "state": "pass",
	      "suggestions": [
	      ],
	      "crname": "example-memcached"
      },
    ]
```

The test image will be executed as a Pod by the scorecard with a restart policy of `never`.

When executing a scorecard test, scorecard will create an environment variable in the test container that holds the name of the CR being tested, this allows the test writer a means of knowing which CR is being created by the scorecard as part of this test execution.

The environment variable created in the test image container will be as follows:
```
CR_NAME=<some CR name>
```

Also, for each test execution, scorecard would also make the
bundle image contents available to test images by mounting
the bundle image contents into a shared emptyDir volume mount
at the following location:

```
/scorecard/bundle
```

This allows tests to have access to all the bundle contents if they 
are required for the test logic.


For the case of executing a test, the bundle image will be passed as a scorecard command flag:
```
operator-sdk scorecard --bundle-image='quay.io/myorg/mytest' --selector=suite=basic
```

A command will need to be built as part of this proposal to allow
end users the ability to pass in the scorecard test files when building
their operator bundle images.

For this, scorecard would need to support bundle images which is included as a User Story in this proposal.  

#### Test Output

Custom tests are expected to produce log output that conforms to the
scorecard v1alpha2 Test Result output.  This allows scorecard to parse
the custom test log output and aggregate the results like any other
scorecard test.

The scorecard expected output would be in JSON, and would match the following definition:  https://github.com/operator-framework/operator-sdk/blob/master/pkg/apis/scorecard/v1alpha2/types.go#L36

The scorecard v1alpha2 Test Result output contains information such as
pass or fail status and suggestions/errors produced by the test that
the end user would want to see in the aggregated scorecard results.


### Custom Test Development Model

The developer of a custom test would have a workflow similar to:
 
 * write a custom test binary in the language of their choice, sample custom test within the SDK repository would be available as a starting point
 * test the custom test binary locally (out-of-cluster) to make sure the output is produced that matches v1alpha2 ScorecardTestResult format
 * create a Pod manifest to run their custom test image, copying a sample Pod manifest from the scorecard examples in the SDK repository
 * when the test binary works as expected, the developer would build and push the test image to a container registry accessible to the scorecard test environment
 * the developer would rebuild their bundle image passing in their scorecard configuration file and their test pod manifests
 * the Basic and OLM test manifests would be added automatically to the bundle image as part of the bundle image build process.  This allows users to run the Basic and OLM tests along with their custom tests.
 * lastly, the developer would run the scorecard passing in their operator bundle image they just built along with any other scorecard flags (e.g. selector).

### Custom Test Packaging and Configuration

In this design proposal, custom tests are Pod manifests in YAML that run a custom test image.  End users would add their test pod manifests and scorecard configuration file into their operator bundle image using a CLI command developed as part of this proposal.

### Scorecard Configuration

Custom test Pod manifests would be accessible from a local disk location to scorecard.  Scorecard would obtain the custom test manifests as part of its bundle image processing since the bundle image is passed into scorecard as a configuration flag.  

Scorecard would assume that the scorecard configuration file and test
manifests would be located in the bundle image at the following location:
```
/bundle/scorecard/.osdk-scorecard.yaml
/bundle/scorecard/tests
```

Test Pod manifests would be configured from within the scorecard configuration file.  The following is a scorecard configuration file example using this scheme:

```
scorecard:
  # Setting a global scorecard option
  output: text
  tests:
    init-timeout: 60
    test-path:
      - "/bundle/scorecard/tests"
  …
```


### Custom Test Execution

Scorecard tests would be executed from the SDK scorecard CLI as is the case today.  Scorecard would execute the tests, capture the results, and provide feedback to the end-user by means of the current scorecard API (e.g. ScorecardTestResult).  Command line flags would allow the end-user a means to specify which tests are executed and upon which test environment to run the tests.  For example:
```
operator-sdk scorecard -o text --selector=suite=isvcustom --bundle-image='some/bundle/image:v0.0.1'
```

Running the above command would have scorecard perform the following:
 
 * fetch the test configuration and test manifests from the bundle image
 * select the tests to run based on the test manifest labels and the selector
 * iterate through each test, 
   * create a unique test namespace
   * create the operator roles and service account in the unique namespace
   * create the operator deployment
   * when the test pod completes, fetch the test image Pod log output for aggregation of test results
   * delete the unique test namespace
 * display the test results 

### Custom Test Environments

Scorecard would allow you to run tests on a local Kube cluster as is the case today.  Provisioning the test cluster is external to the scope of the scorecard itself.  

### Custom Test Labeling

As is the case today, custom tests would allow end users to specify custom labels to each test so that the scorecard could select which tests are run based on a labeling scheme.  Here is an example of user added labels to a custom test step:
```
apiVersion: apps/v1
kind: Pod
metadata:
  name: isvtest1
  labels:
    vendor: someisv
    testdescription: this is a test that does...
    test: isvtest1
    suite: isvcustom
```

The end user could run that test with this command:
```
operator-sdk scorecard -o text --selector=test=isvtest1 --bundle-image='some/bundle/image:v0.0.1'
```

#### Migrating Scorecard Internal Tests to Custom Test Images

With the above custom test capability, there is benefit to Red Hat in migrating the existing scorecard internal tests to be based on the custom test format. 

Today there are 2 internal scorecard tests that make use of the scorecard proxy, with these tests migrating to the custom test format, the scorecard proxy is likely not necessary which would also reduce the overall scorecard implementation complexity.

### Upgrade / Downgrade Strategy

To implement this custom test functionality, additions or changes to the existing scorecard configuration file and command line flags would be necessary and would not be supported in previous scorecard releases.

## Drawbacks


## Ansible and Helm Operator Support

### Ansible Operator Complex Tests

For custom tests, it might be possible to write the custom test image using ansible's callback plugin framework to produce container image log output in the scorecard required format (e.g. v1alpha2).  The complexity of the test might dictate whether this is viable or not, the fallback would be for the custom test to be developed in python, Java, or golang as discussed in this proposal.

### Helm Operator Complex Tests

Writing a custom test image in a Helm chart was not evaluated or pursued in this proposal.  What is likely for a Helm operator is that the operator owner would need to write custom test images using either Python, Java, golang, or some other programming language.


## Alternatives

The alternatives to this design proposal might include:
 * Other existing/open-source test frameworks could be evaluated for use within the SDK for implementing the desired custom test functionality.


## Conclusion

The proposed changes to the scorecard solve the immediate need to support custom or user-provided tests, both simple and complex tests.  However, the longer term implication for scorecard is that all of its tests would evolve to use a single testing format that is far more flexible that what exists today in the current implementation.

The proposed design focuses heavily on separation of concerns, turning scorecard into a test runner essentially, moving test implementations into their own concern (eg. container images).

## Reference Material

[Original Proposal]<https://github.com/operator-framework/operator-sdk/pull/2624>

[kuttl information] <https://github.com/kudobuilder/kuttl>
