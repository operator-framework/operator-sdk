---
title: Scorecard Custom Test Framwork
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
creation-date: 2020-03-04
last-updated: 2020-03-04
status: implementable
---

# Scorecard - Custom Tests


## Release Signoff Checklist

- \[ \] Enhancement is `implementable`
- \[ \] Design details are appropriately documented from clear requirements
- \[ \] Test plan is defined
- \[ \] Graduation criteria for dev preview, tech preview, GA
- \[ \] User-facing documentation is created in [operator-sdk/doc][operator-sdk-doc]

## Summary

This enhancement proposal outlines the SDK scorecard’s design for running of custom tests, specifically tests that are outside of what Red Hat includes by default within the scorecard’s tests.   The proposed design would embed the KUDO test framework into the SDK scorecard and leverage it for an actual implementation, wrapping Kuttl where necessary to provide a good SDK experience to the end-user.

## Motivation

The scorecard needs a robust means of enabling end-users of the SDK to write and execute their own operator tests.   Ideally, the custom test framework would decouple test logic from within the SDK itself, making the SDK and scorecard logic in particular more flexible and maintainable in the long run.

## Goals

 * Document a design for scorecard to execute end-user or custom tests.

 * Document stories/steps required to deliver scorecard custom test functionality.

### Non-Goals

 * This is a design proposal so the sample source code referenced here is not production quality and doesn’t reflect the actual proposed implementation.  Sample tests were constructed for this proposal only to prove out certain aspects of the design.

## Proposal

This design proposal is to integrate the Kuttl test framework into the Operator SDK scorecard specifically for enabling the creation of custom tests that the scorecard could execute and report on.

The design details below break out the components of the test framework offering examples how various types of custom tests could be implemented with this proposed design.

With this proposal, the scorecard will continue to setup and teardown the Operator being tested, as well as creating Custom Resources being tested.

### User Stories

#### Story 1

Scorecard users will be able to construct custom tests and execute them using the scorecard, along with the built-in scorecard tests.

#### Story 2

Scorecard users can develop custom test using the programming language of their choice if that language offers similar support as golang client-go library.

#### Story 3

Scorecard users can execute custom tests as container images.

## Design Details

### Custom Test Definition

* Tests that inspect the state of a Kube resource and compare that state against an expected value.  We’ll call these State Comparison tests.  State comparison tests might include a setup test step.  A test of this type would typically be implemented in a YAML manifest.

* Tests that include logic, operator-specific logic, that determine if an operator is working as expected.  We’ll call these Operator Specific tests.   Operator-specific tests might include a setup test step.  A test of this type might be written in Java or Golang and packaged as a container image.

* Tests that perform static checking of manifests.  We’ll call these Static Manifest tests.  A test of this type might be written in Java or Golang and packaged as a container image.  Static tests like this do not typically require a running Kube cluster but merely examine on-disk operator manifests for correctness.

### Custom Test Components

This proposal outlines the end-user facing components that make up a custom test including:

#### Test Step

A test step is part of a custom test and is meant to setup any Kube resources required for the specific test.  Test steps are coded in a YAML manifest and might include a Pod, Deployment, or Custom Resource manifest.  Test steps are read and processed by the scorecard in order to execute a custom test.  Test steps are read before a test assertion is executed by the scorecard.

Test steps are expected to be provided by the scorecard end user in YAML format.  The path to these test steps would be configured within the scorecard configuration file or command line flags.

#### Test Assertion

A test assertion is the test check that is being performed.  The test assertion is written in a YAML manifest.  The scorecard executes test steps for a custom test prior to executing the test assertion.  The assertion is either true or false, or in the case of scorecard, a pass or fail.

Test assertions are expected to be provided by the scorecard end user in YAML format.  The path to those assertions would be configured within the scorecard configuration file or command line flags.  Test assertions are related and physically organized on disk to match Test Steps.

#### Test Images

Complex tests would require a test container image, that image would be provided by the end user.  The test image would have some common interfaces such as where the scorecard configuration files (e.g. CRs) would be mounted within the container, and how the pass/fail is represented (e.g. Failed or Successful).  There would also be an expected format of the test image log output so that scorecard could process the output log if necessary.

The test image would  adhere to the following common interface:

* Output would be in v1alpha2 ScorecardTestResult format, this is required for the scorecard to parse the test image log output and report the test results to the scorecard user, all scorecard tests would require this output format.

 * Input would be a ConfigMap holding the scorecard configuration such as the CR being tested.  The ConfigMap would follow a naming convention of scorecard-custom-test and would include the CR being tested.  The custom test image could either mount the ConfigMap within its Dockerfile or read it dynamically using an API (e.g.  client-go).

Test image qualified names are specified in Test Steps.

### Custom Test Format
Let's describe what a test would look like in the proposed design.

#### State Comparison Test Format

Custom tests can be written in the form of assertions that are built using YAML.  Assertions are simple tests to see if a particular Kubernetes resource is in a particular state after some period of time.  Examples of an assertion might be a number of requested pods that result from the creation of a particular Custom Resource (CR).  The scorecard executes the assertion, watching for a specific state, and returns a ScorecardTestResult upon timeout.

An example of a State Comparison test (assertion) might look as follows:
```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: example-deployment
status:
  readyReplicas: 3
```

In the above example, this assertion is evaluated as being true or not, this assumes that a Deployment, named example-deployment, with 3 replicas is present on the Kube cluster for example.   The scorecard would execute this assertion as a part of a custom test, the assertion is a YAML manifest.

Test setup steps for a test of this type can be defined using YAML such as:
```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: example-deployment
  labels:
    app: nginx
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
...
```

#### Complex Test Format
Complex tests include logic that is very specific or complex in nature to a given operator.  For example, a database operator that might want to test backup and restore operations that might involve multiple CRs and multiple Kube resources.  
To support this type of test, test logic would be stored within a container image, then the test  essentially becomes the execution of that test container image.  You would then create a test assertion that checks the test pod’s status.   The scorecard would read the Complex test Pod’s log for more detailed information that could be presented to the end-user.

In this scenario, a test setup step would look as follows:
```
apiVersion: v1
kind: Pod
metadata:
  name: custom-test-of-something
  namespace: mynamespace
spec:
  restartPolicy: OnFailure
  containers:
  - name: mytest
    image: someuser/sometestimage:3.6.9-ubi8
    Env:
…
```

Then the test assertion might look as follows:

```
apiVersion: v1
kind: Pod
metadata:
 name: custom-test-of-something
status:
 phase: Successful
```

#### Static Check Test Format

Custom tests that are merely a static check are implemented in a similar fashion as a Complex test, however, they do not require a Kube connection necessarily.  A static check custom test would be started as a Pod, with any required on-disk manifests bundled within a ConfigMap that is available to the static check test Pod.  A test assertion would be used to determine if the static check Pod was successful or not just like the other custom test types.  Static check test logs are contained within the Pod’s log, the scorecard would read the Pod’s logs to display back to the end user.

### Custom Test Assertion Output Format

With all scorecard test types, the test assertion output is translated into a scorecard v1alpha2 ScorecardTestResult structure for consistency in reporting test results.

### Custom Test Packaging and Configuration

In this design proposal, the test assertions are YAML files, test steps are YAML files as well.  Container images holding the custom test logic are expected to be pulled from a registry that is accessible in the Kube test environment that is executing the scorecard tests.

Custom tests and the YAML files that make them up would be accessible via URL or from a local disk location.  Custom test locations would be configured from within the scorecard configuration file.  For example, the scorecard configuration might be as follows to indicate the presence of custom tests from different sources:
```
scorecard:
  # Setting a global scorecard option
  output: json
  plugins:
    - tests:
        init-timeout: 60
        test-paths:
          - "tests/scorecard-tests"
          - "https://some-of-my-tests"
          - "tests/mycustom-tests
cr-manifest:
          - "deploy/crds/cache.example.com_v1alpha1_memcached_cr.yaml"
          - "deploy/crds/cache.example.com_v1alpha1_memcached_cr2.yaml"
  …
```


Here is an on-disk representation of a set of custom tests:
```
tests
├── e2e
│   └── example-test
│       ├── 00-assert.yaml
│       └── 00-install.yaml
├── e2ecomplex
│   └── complex-test
│       ├── 00-assert.yaml
│       └── 00-install.yaml
└── e2estatic
    └── static-test
        ├── 00-assert.yaml
        ├── 00-install.yaml
        ├── 01-assert.yaml
        ├── 01-install.yaml
        └── 02-cleanup.yaml
```

Note:  It might be possible to package custom tests using s2i (source to image)...- Trevor McKay (Red Hat)

Kudo itself has a test configuration file that is used to specify various testing configuration settings.  This file looks like the following example:
```
apiVersion: kudo.dev/v1alpha1
kind: TestSuite
testDirs:
- ./tests/e2e/
startKIND: true
```
Ideally, the scorecard’s current configuration file would be used instead to incorporate any Kudo test configuration settings so that the end user would not have to care about maintaining 2 separate testing configuration files.  The Kuttl project and its API would likely need to support the ability to plug in test configuration settings in order for scorecard to not have to specify a Kuttl specific configuration file.

### Custom Test Execution

Scorecard tests would be executed from the SDK scorecard CLI as is the case today.  Scorecard would execute the tests, capture the results, and provide feedback to the end-user by means of the current scorecard API (e.g. ScorecardTestResult).  Command line flags would allow the end-user a means to specify which tests are executed and upon which test environment to run the tests.  For example:
```
operator-sdk scorecard -o text --selector=custom=test7
```

The scorecard would execute custom tests in the form of YAML manifests (on-disk or network accessible), and container images (via container registry).

### Custom Test Environments

Scorecard would allow you to run tests on a local Kube cluster as is the case today.  Provisioning the test cluster is external to the scope of the scorecard itself.  For example, some might want to run a scorecard on a provisioned environment such as kind (kube in docker).  

### Custom Test Labeling
As is the case today, custom tests would allow end users to specify custom labels to each test so that the scorecard could select which tests are run based on a labeling scheme.  Here is an example of user added labels to a custom test step:
```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deployment
  labels:
    vendor: redhat
    test: mycustomtest
    testsuite: custom
spec:
  replicas: 4
```

The end user could run that test with this command:
```
operator-sdk scorecard -o text --selector=test=mycustomtest
```
The scorecard would look at on-disk custom tests as well as tests referenced by URL and perform label selection/matching to determine which tests would be executed.

#### Sample Scorecard Usage

Here is how an SDK user would work with scorecard after this enhancement is implemented:

 * Develop Custom Test Steps
 * Develop Custom Test Assertions
 * Configure Scorecard to Locate Custom Tests
 * Execute Scorecard Specifying Custom Test Label(s)
 * Scorecard Aggregates Custom Test Output/Results
 * Scorecard Displays Test Results

#### Phase 2

For consideration, with the above custom test capability, there is benefit to Red Hat in migrating the existing scorecard internal tests to be based on the custom test format.  For this document, I’ve called this work, Phase 2, and is optional but worth consideration.  

Today there are 2 internal scorecard tests that make use of the scorecard proxy, with these tests migrating to the custom test format, the scorecard proxy is likely not necessary which would also reduce the overall scorecard implementation complexity.

#### Building the Proposed Design

The proposed design was influenced greatly by the KUDO testing framework in various concepts, however, the proposed design doesn’t necessarily have to include KUDO as a dependency.  A decision of whether or not to incorporate KUDO testing APIs within the Red Hat solution needs to be reached.  Some pro’s and con’s to using KUDO testing frameworks are listed below.  If KUDO’s testing framework is not incorporated, then a custom built set of equivalent capabilities would need to be developed.
The advantage of using [Kuttl](https://github.com/kudobuilder/kuttl) as the test framework in scorecard would include:

 * Pre-built, works, community, evolving.
 * Ready to work with, code exists and mostly works, a starting point exists.
 * Handles the basic test case, that being to test Kube resource state
 * Written in golang, making integration into scorecard a possibility.


### Test Plan

**Note:** *Section not required until targeted at a release.*

### Upgrade / Downgrade Strategy

To implement this custom test functionality, additions or changes to the existing scorecard configuration file and command line flags would be necessary and would not be supported in previous scorecard releases.

## Implementation History

 * Proposal initiated in March 2020.

## Drawbacks

Drawbacks of using Kuttl within this proposed design might include:

 * Immaturity of Kuttl and KUDO in general, integration into Operator SDK will require Redhat to possibly contribute upstream to the Kuttl project itself in order to meet the needs of the Operator SDK.
  * <https://github.com/kudobuilder/kuttl/issues/16#issuecomment-589242819>
  * <https://github.com/kudobuilder/kuttl/issues/12>
  * <https://github.com/kudobuilder/kudo/blob/master/keps/0008-operator-testing.md>
  * <https://github.com/kudobuilder/kuttl/pull/5>

 * Dependency on a community project for enhancements or stability of APIs
 * Doesn’t do much in terms of complex testing
 * Needs some kind of regex capability for a more robust YAML testing capability
 * Will require integration work to include it into the scorecard code.
 * KUDO doesn’t currently make it easy to get the test Pod log output, this is useful to determine why a test might have failed, without it being built-into Kudo, the scorecard would need to read the Pod logs to present back to the end user.

## Alternatives

The alternatives to this design proposal might include:
 * A custom built test framework be designed and developed solely for the purpose of implementing Operator SDK scorecard custom tests.
 * Other existing/open-source test frameworks could be evaluated for use within the SDK for implementing the desired custom test functionality.


## Conclusion

The proposed changes to the scorecard solve the immediate need to support custom or user-provided tests, both simple and complex tests.  However, the longer term implication for scorecard is that all of its tests would evolve to use a single testing format that is far more flexible that what exists today in the current implementation.

The proposed design focuses heavily on separation of concerns, turning scorecard into a test runner essentially, moving Kube provisioning/setup responsibilities outside of the scorecard, and moving test implementations into their own concern (YAML manifests and/or container images).

**The proposed design would be implemented by means of integrating the Kuttl library into the SDK scorecard codebase.**

## Reference Material

<https://github.com/kudobuilder/kudo>

<https://github.com/kudobuilder/kuttl>

The testing framework used within the KUDO project is being pulled out into its own repo, and named Kuttl.  This framework has some appealing features in it that could be useful for a scorecard implementation.  Those being:

 * Simple yaml assertion definition for tests that check Kube resource state
 * CLI that lets you run tests and view pass/fail results.
 * On-disk structure that defines tests and test suites.

### Sample Code

To test out some of the recommendations of this proposal, I constructed some real examples using KUDO test to execute tests against a SDK-generated operator that I wrote some time ago.  This operator has zero KUDO dependencies.

These test container images below were implemented in golang, but could be implemented in any language that has a capability similar to client-go (e.g. talk to the Kube API).  Note that these test images do not make use of controller-runtime in their current form.

#### Sample Static Complex Test

To test out this proposal I constructed the following Kudo tests:
<https://github.com/jmccormick2001/my-static-test>
This test implements a OLM bundle validation test similar to that of the current scorecard OLM Bundle Validation test.  This test harness consists of a container image that examines the bundle, provided by means of a ConfigMap, and validates it using the operator-framework/api validation API.  If there are no errors, the the container image exits normally, if not, it will exit with a failure exit code (1).
There is a static test constructed that is run on a sample operator, the rqlite-operator that I wrote previously as a learning tool.  The static test harness is found here:
<https://github.com/jmccormick2001/rqlite-operator/tree/master/tests/e2estatic/static-test>

The test steps include:

 * Deleting the test Pod if it exists
 * Deleting the bundle ConfigMap if it exists
 * Creating the bundle ConfigMap in the ‘rq’ namespace
 * Testing if a ConfigMap “my-static-test” has been created in the ‘rq’ namespace.
 * Creating the test Pod, passing in the name of the ConfigMap that contains the bundle directory, also passing in the Pod namespace
 * Checks to see if the test Pod is in a ‘Succeeded’ phase, this is the assertion that tells us if the bundle validation passed with no errors or not.
 * Finally, a cleanup step that removes the bundle ConfigMap



#### Sample Complex Test

There is an example complex test located here:
<https://github.com/jmccormick2001/rqlite-operator/tree/master/tests/e2ecomplex/complex-test>

The example test container is implemented here:
<https://github.com/jmccormick2001/my-custom-test>
The complex test example evaluates a CR’s number of nodes to the number of expected by the test, if they don’t match, the test container image will pass a failure exit code.

The test consists of the following steps:

 * Create a test Pod, passing in the expected number of Nodes, and the Custom Resource name to examine.
 * Assert that the Pod succeeded

#### Sample State Comparison Test

There is an example of a very simple YAML based state comparison test here:
<https://github.com/jmccormick2001/rqlite-operator/tree/master/tests/e2e/example-test>
This test consists of the following steps:

 * Create a rqcluster Custom Resource
 * Assert that a Pod was created and in Running state, that matches the expected Pods normally created when the Custom Resource is created.


[operator-sdk-doc]:  ../../doc
