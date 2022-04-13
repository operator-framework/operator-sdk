---
title: Scorecard Custom Test Framework
authors:
  - "@jemccorm"
reviewers:
  - "@joelandford"
  - "@zeus"
approvers:
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

With this proposal, the scorecard will change as follows:

 * scorecard will not start the operator being tested
 * scorecard will not create Custom Resources on behalf of test
 * scorecard will execute tests as Pods based on scorecard configuration
 * scorecard will require a valid bundle image or on-disk bundle to determine what information will be passed to scorecard tests
 * scorecard will determine test configurations by a YAML/DSL file that is either passed on the command line or included in the bundle contents
 * scorecard will only run OLM tests if OLM manifests are found in the bundle
 * scorecard command syntax will look as follows:
   * operator-sdk scorecard --dsl <point-to-yaml> --bundle <bundle-path/image> -l <selector>

### User Stories

#### Story 1

I am as an Operator developer, I'd like to be able to construct custom tests and execute them via the scorecard.

#### Story 2

Scorecard will include an example custom test that end users could use as a reference when writing their custom tests.

#### Story 3

Scorecard internal tests will be converted to the custom test format and therefore be externalized from the scorecard binary itself.

#### Story 4

Scorecard would process tests based on an on-disk directory or bundle image that contains the scorecard configuration.

#### Story 5 

- I am as an Operator developer, would like to be able to create tests using the same Assert syntax adopted by [Kuttl](https://github.com/kudobuilder/kuttl) and check its results aggregated within scorecard results. 

**Implementation Notes**

The Scorecard would support the creation of a Kuttl test image and then execute tests via Kuttl, however, it would be transparent for users since its results will be integrated into scorecard results.

## Design Details

### Glossary

 * test image - this is a container image built by end-users to contain a custom test implementation
 * test configuration (DSL) - this is a file that contains the test definitions and labels which determine what tests are run and how they are configured
 * Custom Test Definition - A custom test is any sort of test logic required, written in any programming language that satisfies the scorecard test requirements for output format and execution.

#### Test Image

A custom test is based on a container image, that image is created by
the end-user using whatever tools they want to use.

The test image must produce container log output that conforms to the scorecard Test Result output.  The output is a single Test Result, so a single run of a test image should execute and output the result for exactly one test.  Here is a sample of test output in the required format:
```
      {
	      "name": "Spec fields with descriptors",
	      "description": "All spec fields have matching descriptors in the CSV",
	      "state": "fail",
	      "suggestions": [
		"Add a spec descriptor for size"
	      ],
	      "crname": "example-memcached"
      } 
```

The test image will be executed as a Pod by the scorecard with a restart policy of `never`.

Also, for each test execution, scorecard would also make the
bundle image contents available to test images by mounting
the bundle image contents into the following location inside the test pod:

```
/scorecard/bundle/
```

This allows tests to have access to all the bundle contents if they 
are required for the test logic.  The scorecard does not make assumptions about what the test does with this proposed design.  The test could for example:

 * not use the CR's at all (e.g. doing a static bundle validation)
 * use the CRs in the CSVs alm-examples annotation (e.g. to verify they validate the associated CRD schema)
 * use some other CRs built (or maybe mounted) into the test image 

For the case of executing a test, the bundle image will be passed as a scorecard command flag:
```
operator-sdk scorecard --bundle='quay.io/myorg/mytest' --selector=suite=basic
```

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
 * scorecard creates a scorecard DSL config file for their test image
 * when the test binary works as expected, the developer would build and push the test image to a container registry accessible to the scorecard test environment
 * scorecard could build a bundle image that contains the scorecard DSL config
 NOTE:  the Basic and OLM test manifests would be added automatically to the bundle image as part of the bundle image build process.  This allows users to run the Basic and OLM tests along with their custom tests.
 * the developer would run the scorecard passing in their operator bundle image they just built along with any other scorecard flags (e.g. selector).

The case of a developer implementing a kuttl test would be as follows:
 * developer places kuttl test assets into a kuttl test directory known to scorecard
 * scorecard creates the scorecard DSL config file
 * scorecard runs the tests which are based on a scorecard kuttl test image
 * if the test results meet their goals, then scorecard could build a bundle image that contains the kuttl tests in a distributable format (e.g. bundle image)

### Custom Test Packaging and Configuration

In this design proposal, custom tests are container images, described in the scorecard DSL yaml file.  End users would add their scorecard DSL yaml file into their operator bundle image or pass it into scorecard using a command line flag.  Custom test images are stored in a container registry outside the control of scorecard.  The scorecard DSL command flag would take precedence over the scorecard DSL config file found in the bundle image if specified.

### Scorecard Configuration

The scorecard DSL yaml file would be accessible from a local disk location to scorecard or from within a bundle image. 

Scorecard would assume that the scorecard DSL yaml file would be located in the bundle image at the following location:
```
/scorecard/config.yaml
```

The scorecard DSL format is yet to be finalized but it would need to include configuration settings for each test including labels.  Here is a sample of what the scorecard DSL configuration would look like:
```
tests:
# here is an example of a pair of custom tests that an ISV might implement
- image: quay.io/someuser/customtest1:v0.0.1
  description: some ISV test
  labels:
    suite: custom
    test: customtest1
- image: quay.io/someuser/customtest2:v0.0.1
  description: some ISV test
  labels:
    test: customtest2
# here is an example of the scorecard basic tests
- image: quay.io/operator-framework/scorecard-basictests:v0.0.1
  description: redhat - check spec
  entrypoint: checkspec
  labels:
    suite: basic
    test: basic-check-spec-test
- image: quay.io/operator-framework/scorecard-basictests:v0.0.1
  description: redhat - check status
  entrypoint: checkstatus
  labels:
    suite: basic
    test: basic-check-status-test
# here is an example of the scorecard olm tests
- image: quay.io/operator-framework/scorecard-olmtests:v0.0.1
  description: redhat - CRDs have validation
  entrypoint: crdvalidation
  labels:
    suite: olm
    test: olm-crds-have-validation-test
- image: quay.io/operator-framework/scorecard-olmtests:v0.0.1
  description: redhat - CRDs have resources
  entrypoint: crdhasresources
  labels:
    suite: olm
    test: olm-crds-have-resources-test
- image: quay.io/operator-framework/scorecard-olmtests:v0.0.1
  description: redhat - spec descriptors
  entrypoint: crdspecdescriptors
  labels:
    suite: olm
    test: olm-spec-descriptors-test
- image: quay.io/operator-framework/scorecard-olmtests:v0.0.1
  description: redhat - status descriptors
  entrypoint: crdstatusdescriptors
  labels:
    suite: olm
    test: olm-status-descriptors-test
- image: quay.io/operator-framework/scorecard-olmtests:v0.0.1
  description: redhat - bundle validation
  entrypoint: bundlevalidation
  labels:
    suite: olm
    test: olm-bundle-validation-test
# here is an example of the scorecard kuttl tests
- image: quay.io/operator-framework/scorecard-kuttltests:v0.0.1
  description: redhat - kuttl tests
  labels:
    suite: kuttl
```

Users could override the default test images shipped with scorecard
by adding scorecard command flags as follows:
```
operator-sdk scorecard --olm-image=quay.io/operator-framework/scorecard-olmtests:dev
operator-sdk scorecard --basic-image=quay.io/operator-framework/scorecard-basictests:dev
operator-sdk scorecard --kuttl-image=quay.io/operator-framework/scorecard-kuttltests:dev
```

### Custom Test Execution

Scorecard tests would be executed from the SDK scorecard CLI as is the case today.  Scorecard would execute the tests, capture the results, and provide feedback to the end-user by means of the current scorecard API (e.g. ScorecardTestResult).  Command line flags would allow the end-user a means to specify which tests are executed and upon which test environment to run the tests.  For example:
```
operator-sdk scorecard -o text --selector=suite=isvcustom --bundle='some/bundle/image:v0.0.1'
```

Running the above command would have scorecard perform the following:
 
 * fetch the scorecard DSL yaml configuration from the bundle image
 * select the tests to run based on the scorecard DSL configuration and the selector
 * iterate through each test, 
   * construct a test Pod 
   * create the test Pod
   * when the test pod completes, fetch the test image Pod log output for aggregation of test results
 * display the test results 

### Custom Test Environments

Scorecard would allow you to run tests on a local Kube cluster as is the case today.  Provisioning the test cluster is external to the scope of the scorecard itself.  Note, that with this proposal, scorecard would not be responsible for installing OLM or depend on OLM being installed in order to run. 


#### Migrating Scorecard Internal Tests to Custom Test Images

With the above custom test capability, there is benefit to Red Hat in migrating the existing scorecard internal tests to be based on the custom test format. 

Today there are 2 internal scorecard tests that make use of the scorecard proxy, with these tests migrating to the custom test format, the scorecard proxy is likely not necessary which would also reduce the overall scorecard implementation complexity.

### Upgrade / Downgrade Strategy

The changes to scorecard are large and would require a migration off
of the current scorecard.

## Drawbacks


## Ansible and Helm Operator Support

### Ansible Operator Complex Tests

For custom tests, it might be possible to write the custom test image using ansible's callback plugin framework to produce container image log output in the scorecard required format (e.g. v1alpha2).  The complexity of the test might dictate whether this is viable or not, the fallback would be for the custom test to be developed in python, Java, or golang as discussed in this proposal.

### Helm Operator Complex Tests

Writing a custom test image in a Helm chart was not evaluated or pursued in this proposal.  What is likely for a Helm operator is that the operator owner would need to write custom test images using either Python, Java, golang, or some other programming language.


## Alternatives

The alternatives to this design proposal might include:
 * Other existing/open-source test frameworks could be evaluated for use within the SDK for implementing the desired custom test functionality.
 * Other test frameworks within Redhat being planned or within use could be evaluated.


## Conclusion

The proposed changes to the scorecard solve the immediate need to support custom or user-provided tests, both simple and complex tests.  However, the longer term implication for scorecard is that all of its tests would evolve to use a single testing format that is far more flexible that what exists today in the current implementation.

The proposed design focuses heavily on separation of concerns, turning scorecard into a test runner essentially, moving test implementations into their own concern (eg. container images).

## Reference Material
([Original Proposal](https://github.com/operator-framework/operator-sdk/pull/2624))
([kuttl information](https://github.com/kudobuilder/kuttl))
