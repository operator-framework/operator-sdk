# Simple YAML Defined Test Plugin for Operator Scorecard

Implementation Owner: AlexNPavel

Status: Draft

[Background](#Background)

[Goals](#Goals)

[Design overview](#Design_overview)

## Background

The operator scorecard has recently implemented a plugin system that allows the scorecard to be easily extended. This is a proposal to provide users
with a new user-defined scorecard test that can allow users to perform more specific tests on their operator compared to the Basic and OLM test plugins.

## Goals

- Implement a simple scorecard plugin that allows a user to define tests catered specifically to their operator

## Design overview

### YAML Defined Test

The new plugin system will allow users to define simple tests in YAML. In these tests, a user can specify what values the fields of a resource should have after creation of all resources has completed. For example, this definition runs a similar test to the memcached-operator scale test from the SDK's e2e test:

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
start with `scorecard_function_`, we do a simple match (like `status/readyReplicas: 4`). Another possible scorecard function would be a regex checker to
make sure that a string matches an expected pattern.
