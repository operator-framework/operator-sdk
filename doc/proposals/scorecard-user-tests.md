# User Defined Tests for the Operator Scorecard

Implementation Owner: AlexNPavel

Status: Draft

[Background](#Background)

[Goals](#Goals)

[Design overview](#Design_overview)

[User facing usage](#User_facing_usage)

[Observations and open questions](#Observations_and_open_questions)

## Background

The operator scorecard is intended to allow users to run a generic set of tests on their operators. Some simple checks can be performed, but more complicated
tests verifying that the operator actually works are not possible to do in a generic way. This leads to some of the functional tests in the current scorecard
implementation to be inaccurate or too insufficient to be useful. For more useful functional scorecard tests, we need to allow some basic user input for tests.

## Goals

- Implement user-defined scorecard tests
- Replace existing "Operator actions are reflected in status" and "Writing into CRs has effects" tests with user-defined tests

## Design overview

### Basic YAML Defined Test

A new basic testing system would be added where a user can simply define various aspects of a test. For example, this definition runs a similar test to the memcached-operator scale test from the SDK's e2e test:

```yaml
functional_tests:
  - cr: "deploy/crds/cache_v1alpha1_memcached_cr.yaml"
    expected:
      resources:
        - apiVersion: apps/v1
          kind: Deployment
          metadata:
            name: example-memcached
          status:
            readyReplicas: 3
          spec:
            template:
              spec:
                containers:
                  - image: memcached:1.4.36-alpine
      status:
        scorecard_function_length:
          nodes: 3
    modifications:
      - spec:
          size: 4
        expected:
          resources:
            - kind: Deployment
              name: "example_memcached"
              status:
                readyReplicas: 4
          status:
            scorecard_function_length:
              nodes: 4
```

This is what the golang structs would look like, including comments describing each field:

```go
// Struct containing a user defined test. User passes tests as an array using the `functional_tests` viper config
type UserDefinedTest struct {
    // Path to cr to be used for testing
    CRPath string `mapstructure:"cr"`
    // Expected resources and status
    Expected Expected `mapstructure:"expected"`
    // Sub-tests modifying a few fields with expected changes
    Modifications []Modification `mapstructure:"modifications"`
}

type Expected struct {
	// Resources expected to be created after the operator reacts to the CR
	Resources []map[string]interface{} `mapstructure:"resources"`
	// Expected values in CR's status after the operator reacts to the CR
	Status map[string]interface{} `mapstructure:"status"`
}

// Modifications specifies a spec field to change in the CR with the expected results
type Modification struct {
    // a map of the spec fields to modify
    Spec map[string]interface{} `mapstructure:"spec"`
    // Expected resources and status
    Expected Expected `mapstructure:"expected"`
}
```

For `Status` and `Resources` fields, we can implement a bit of extra computation instead of simple string checking. For instance,
in the memcached-operator test, we should expect that the length of the `nodes` field (which is an array) has a certain length. To implement functions like
these, we can create some functions for these checks that are prepended by `scorecard_function_` and take an array of objects. For instance, in the above
example, `scorecard_function_length` would check that each field listed under it matches the specified length (like `nodes: 4`). If the yaml key does not
start with `scorecard_function_`, we do a simple match (like `status/readyReplicas: 4`).

This design would allow us to replace the old "Operator actions are reflected in status" (which would be tested by the `expected/status` check) and
"Writing into CRs has an effect" (which would be tested by the `expected/resources` check) tests.

### Plugin System

In order to increase the flexibility of the user defined tests and allow users to implement more complex E2E style tests for scorecard,
the user-defined tests will be implemented via a plugin system. Users will specify the path of the script they wish to run as well
as environment variable to set for the command. The command would then print out the result as JSON to stdout. Here is an example:

```yaml
user_defined_tests:
- path: "scorecard/simple-scorecard.sh"
  env:
    - CONFIG_FILE: "scorecard/simple-scorecard.yaml"
- path: "scorecard/go-test.sh"
  env:
    - ENABLE_SCORECARD: true
    - NAMESPACED_MANIFEST: "deploy/namespaced_init.yaml"
    - GO_TEST_FLAGS: "-parallel=1"
```

The above is an example of a user-defined test suite with 2 tests: the new simple scorecard tests described above and a test
built using the Operator SDK's test framework that's been modified to be able to output results in the standardized JSON output.

Below is an example of what the JSON output of a test would look like:

```json
{
  "actions_reflected_in_status": {
    "description": "The operator correctly updates the CR's status",
    "earned": 2,
    "maximum": 3,
    "suggestions": [
      {
        "message": "Expected 4 items in status/nodes but found 2"
      }
    ],
    "errors": []
  },
  "my_custom_tests": {
    "description": "This test verifies that the created service is running correctly",
    "earned": 1,
    "maximum": 1,
    "suggestions": [],
    "errors": []
  },
}
```

This JSON output would make it simple for others to create scorecard plugins while keeping it simple for the scorecard
to parse and integrate with the other tests. The above JSON design is based on the `TestResult` type that will be implemented
by PR [#994](https://github.com/operator-framework/operator-sdk/pull/994).
