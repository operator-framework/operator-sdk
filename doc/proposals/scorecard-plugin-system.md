# Plugin System for the Operator Scorecard

Implementation Owner: AlexNPavel

Status: Draft

[Background](#Background)

[Goals](#Goals)

[Design overview](#Design_overview)

## Background

The operator scorecard is intended to allow users to run a generic set of tests on their operators. The scorecard currently only has
built-in tests, and it would be beneficial to allow a simple way to add or remove various tests that the scorecard can run. This proposal
outlines a plugin system that would allow us and users to dynamically add new tests without having to compile them into the scorecard/SDK
binary.

## Goals

- Implement a configurable plugin based scorecard test system

## Design Overview

### Plugin System

In order to increase the flexibility of the user defined tests and allow users to implement more complex E2E style tests for scorecard,
the user-defined tests will be implemented via a plugin system. Users would put executable files (etiher scripts or binaries) in a directory
in the project root, for example `<root>/scorecard/bin` (the path can be configured via a flag). The scorecard would run all exectuable files
sequentially and each plugin is expected to print out the result as JSON to stdout. If a plugin has a fatal error or does not return a valid JSON
result, the scorecard will have a default failure JSON result that specifies that the binary/script failed to run along with what the executable printed
to stdout.

The JSON output will be reusing the Kubernetes API for marshalling and unmarshalling. This would allow us to have a standardized `TypeMeta` that will allow
us to update the way we define tests and results in the future with proper versioning. Below is an example of what the JSON output of a test would look like:

Go structs:

```go
type ScorecardTest struct {
    metav1.TypeMeta `json:",inline"`
    // Spec describes the attributes for the test.
    Spec *ScorecardTestSpec `json:"spec"`

    // Status describes the current state of the test and final results.
    // +optional
    Status *ScorecardTestResults `json:"results,omitempty"`
}

type ScorecardTestSpec struct {
    // TestInfo is currently used for ScorecardTestSpec.
    TestInfo `json:",inline"`
}

type ScorecardTestResults struct {
    // Log contains the scorecard's current log.
    Log string `json:"log"`
    // Results is an array of ScorecardResult for each suite of the curent scorecard run.
    Results []ScorecardResult `json:"results"`
}

// ScorecardResult contains the combined results of a suite of tests
type ScorecardResult struct {
    // Error is the number of tests that ended in the Error state
    Error       int               `json:"error"`
    // Pass is the number of tests that ended in the Pass state
    Pass        int               `json:"pass"`
    // PartialPass is the number of tests that ended in the PartialPass state
    PartialPass int               `json:"partial_pass"`
    // Fail is the number of tests that ended in the Fail state
    Fail        int               `json:"fail"`
    // TotalTests is the total number of tests run in this suite
    TotalTests  int               `json:"total_tests"`
    // TotalScore is the total score of this quite as a percentage
    TotalScore  int               `json:"total_score_percent"`
    // Tests is an array containing a json-ified version of the TestResults for the suite
    Tests       []*JSONTestResult `json:"tests"`
}

// JSONTestResult is a simplified version of the TestResult that only include the Name and Description of the Test field in TestResult
type JSONTestResult struct {
    // State is the final state of the test
    State         State
    // Name is the name of the test
    Name          string
    // Description describes what the test does
    Description   string
    // EarnedPoints is how many points the test received after running
    EarnedPoints  int
    // MaximumPoints is the maximum number of points possible for the test
    MaximumPoints int
    // Suggestions is a list of suggestions for the user to improve their score (if applicable)
    Suggestions   []string
    // Errors is a list of the errors that occured during the test (this can include both fatal and non-fatal errors)
    Errors        []error
}

// State is a type used to indicate the result state of a Test.
type State string

const (
    // UnsetState is the default state for a TestResult. It must be updated by UpdateState or by the Test.
    UnsetState State = "unset"
    // PassState occurs when a Test's ExpectedPoints == MaximumPoints.
    PassState State = "pass"
    // PartialPassState occurs when a Test's ExpectedPoints < MaximumPoints and ExpectedPoints > 0.
    PartialPassState State = "partial_pass"
    // FailState occurs when a Test's ExpectedPoints == 0.
    FailState State = "fail"
    // ErrorState occurs when a Test encounters a fatal error and the reported points should not be considered.
    ErrorState State = "error"
)
```

JSON output for `ScorecardResult` object (for the initial `v1alpha1` of the scorecard test objects):

```json
{
    "error": 0,
    "pass": 1,
    "partial_pass": 1,
    "fail": 0,
    "total_tests": 2,
    "total_score_percent": 71,
    "tests": [
        {
            "state": "partial_pass",
            "name": "Operator Actions Reflected In Status",
            "description": "The operator updates the Custom Resources status when the application state is updated",
            "earnedPoints": 2,
            "maximumPoints": 3,
            "suggestions": [
                {
                    "suggestion": "Operator should update status when scaling cluster down"
                }
            ],
            "errors": []
        },
        {
            "state": "pass",
            "name": "Verify health of cluster",
            "description": "The cluster created by the operator is working properly",
            "earnedPoints": 1,
            "maximumPoints": 1,
            "suggestions": [],
            "errors": []
        }
    ]
}
```

This JSON output would make it simple for others to create scorecard plugins while keeping it simple for the scorecard
to parse and integrate with the other tests. Each plugin would be considered a separate suite, and the full result of the scorecard
would be a list of `ScorecardResult`s.
