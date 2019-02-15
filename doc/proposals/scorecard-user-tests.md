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

A new section would be added to the scorecard config file called `functional_tests` where a user can define various aspects of a test. For example, this definition runs a similar test to the memcached-operator scale test from the SDK's e2e test:

```yaml
functional_tests:
  - cr: "deploy/crds/cache_v1alpha1_memcached_cr.yaml"
    expected:
      resources:
        - apiVersion: apps/v1
          kind: Deployment
          name: "example-memcached"
          fields:
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
        - size: 4
        expected:
          resources:
            - kind: Deployment
              name: "example_memcached"
              fields:
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
    Resources []ExpectedResource `mapstructure:"resources"`
    // Expected values in CR's status after the operator reacts to the CR
    Status map[string]interface{} `mapstructure:"status"`
}

// Struct containing a resource and its expected fields
type ExpectedResource struct {
    // (if set) Namespace of resource
    Namespace string `mapstructure:"namespace"`
    // APIVersion of resource
    APIVersion string `mapstructure:"apiversion"`
    // Kind of resource
    Kind string `mapstructure:"kind"`
    // Name of resource
    Name string `mapstructure:"name"`
    // The fields we expect to see in this resource
    Fields map[string]interface{} `mapstructure:"fields"`
}

// Modifications specifies a spec field to change in the CR with the expected results
type Modification struct {
    // a map of the spec fields to modify
    Spec map[string]interface{} `mapstructure:"spec"`
    // Expected resources and status
    Expected Expected `mapstructure:"expected"`
}
```

For `Status` fields and `ExpectedResource.Fields`, we can implement a bit of extra computation instead of simple string checking. For instance,
in the memcached-operator test, we should expect that the length of the `nodes` field (which is an array) has a certain length. To implement functions like
these, we can create some functions for these checks that are prepended by `scorecard_function_` and take an array of objects. For instance, in the above
example, `scorecard_function_length` would check that each field listed under it matches the specified length (like `nodes: 4`). If the yaml key does not
start with `scorecard_function_`, we do a simple match (like `status.readyReplicas: 4`).

This design would allow us to replace the old "Operator actions are reflected in status" (which would be tested by the `expected/status` check) and
"Writing into CRs has an effect" (which would be tested by the `expected/resources` check) tests.

## User facing usage (if needed)

We would require a user to provide a config file with the `functional_tests` field run the 2 tests mentioned above. If not provided, those tests would be marked
as failed (`0/1 points`). To make sure a user knows how to design their tests to get a good score in the scorecard, we would need detailed docs.
