# Operator Scorecard

The Operator Scorecard is a testing utility included in the `operator-sdk` binary that guides users towards operator best practices
by checking the correctness of their operators and CSVs. While the Scorecard is in an early
stage of development, it will gain more functionality and stabilize over time.

## How It Works

The scorecard works by creating all resources required by CRs and the operator. For the operator
deployment, it also adds another container to the operator's pod that is used to record calls to the API server,
which are analyzed by the scorecard for various tests. The scorecard will also look at some of the fields in the
CR object itself for some of the tests.

As of `v0.8.0` scorecard also supports plugins. This allows external developers to extend the functionality of the scorecard and add additional tests.

## Requirements

- An operator made using the `operator-sdk` or an operator that uses a config getter that supports reading from the `KUBECONFIG` environment variable (such as the `clientcmd` or `controller-runtime` config getters). This is required for the scorecard proxy to work correctly.
- Resource manifests for installing/configuring the operator and custom resources (see the [Writing E2E Tests][writing-tests] doc for more information on the global and namespaced manifests).
- (OLM tests only) A CSV file for your operator.

## Running the tests

The scorecard currently uses a large amount of flags to configure the scorecard tests. You can see
these flags in the `scorecard` subcommand help text, or in the [SDK CLI Reference][cli-reference] doc. Here, we will highlight a few important
flags:

- `--output`, `-o` - this flag is used to changed the output format. The default is a `human-readable` format with streaming logs. The other format is `json`, which is output in the JSON schema used for plugins defined later in this document.
- `--cr-manifest` - this is a required flag for the scorecard. This flag must point to the location of the manifest(s) you want to test. You can specify multiple CR manifest by specifying this flag multiple times.
- `--csv-path` - this flag is required if the OLM tests are enabled (the tests are enabled by default). This flag must point to the location of the operators' CSV file.
- `--namespaced-manifest` - if set, this flag must point to a manifest file with all resources that run within a namespace. By default, the scorecard will combine `service_account.yaml`, `role.yaml`, `role_binding.yaml`, and `operator.yaml` from the `deploy` directory into a temporary manifest to use as the namespaced manifest.
- `--global-manifest` - if set, this flag must point to all required resources that run globally (not namespaced). By default, the scorecard will combine all CRDs in the `deploy/crds` directory into a temporary manifest to use as the global manifest.
- `--namespace` - if set, which namespace to run the scorecard tests in. If it is not set, the scorecard will use the default namespace of the current context set in the kubeconfig file.
- `--olm-deployed` - indicates that the CSV and relevant CRD's have been deployed onto the cluster by the [Operator Lifecycle Manager (OLM)][olm]. This flag cannot be used in conjunction with `--namespaced-manifest` or `--global-manifest`. See the [CSV-only tests](#running-the-scorecard-with-a-deployed-csv) section below for more details.

To run the tests, simply run the `scorecard` subcommand from your project root with the flags you want to
use. For example:

```console
$ operator-sdk scorecard --cr-manifest deploy/crds/app_operator_cr.yaml --csv-path deploy/app_operator-0.0.2.yaml
```

## Config File

The scorecard supports the use of a config file instead of or in addition to flags for configuration. By default, the scorecard will look
for a file called `.osdk-scorecard` with either a `.yaml`, `.json`, or `.toml` file extension. You can also
specify a different config file with the `--config` flag. The configuration options in the config file match the flags.
For instance, for the flags `--cr-manifest "deploy/crds/cache_v1alpha1_memcached_cr.yaml" --cr-manifest "deploy/crds/cache_v1alpha1_memcached2_cr.yaml" --init-timeout 60 --csv-path "deploy/olm-catalog/memcached-operator/0.0.2/memcached-operator.v0.0.2.clusterserviceversion.yaml"`, the corresponding yaml config file would contain:

```yaml
cr-manifest:
  - "deploy/crds/cache_v1alpha1_memcached_cr.yaml"
  - "deploy/crds/cache_v1alpha1_memcached2_cr.yaml"
init-timeout: 60
csv-path: "deploy/olm-catalog/memcached-operator/0.0.2/memcached-operator.v0.0.2.clusterserviceversion.yaml"
```

The hierarchy of config methods from highest priority to least is: flag->file->default.

The config file support is provided by the `viper` package. For more info on how viper
configuration works, see [`viper`'s README][viper].

## What Each Builtin Test Does

There are 8 builtin tests the scorecard can run. If multiple CRs are specified, the test environment is fully cleaned up after each CR so each CR gets a clean testing environment.

### Basic Operator

#### Spec Block Exists

This test checks the Custom Resource(s) created in the cluster to make sure that all CRs have a spec block. This test
has a maximum score of 1.

#### Status Block Exists

This test checks the Custom Resource(s) created in the cluster to make sure that all CRs have a status block. This
test has a maximum score of 1.

#### Writing Into CRs Has An Effect

This test reads the scorecard proxy's logs to verify that the operator is making `PUT` and/or `POST` requests to the
API server, indicating that it is modifying resources. This test has a maximum score of 1.

### OLM Integration

#### Provided APIs have validation

This test verifies that the CRDs for the provided CRs contain a validation section and that there is validation for each
spec and status field detected in the CR. This test has a maximum score equal to the number of CRs provided via the `--cr-manifest` flag.

#### Owned CRDs Have Resources Listed

This test makes sure that the CRDs for each CR provided via the `--cr-manifest` flag have a `resources` subsection in the [`owned` CRDs section][owned-crds] of the CSV. If the
test detects used resources that are not listed in the resources section, it will list them in the suggestions at the end of the test.
This test has a maximum score equal to the number of CRs provided via the `--cr-manifest` flag.

#### CRs Have At Least 1 Example

This test checks that the CSV has an [`alm-examples` annotation][alm-examples] for each CR passed to the `--cr-manifest` flag in its metadata. This test has a maximum score
equal to the number of CRs provided via the `--cr-manifest` flag.

#### Spec Fields With Descriptors

This test verifies that every field in the Custom Resources' spec sections have a corresponding descriptor listed in
the CSV. This test has a maximum score equal to the total number of fields in the spec sections of each custom resource passed in via the `--cr-manifest` flag.

#### Status Fields With Descriptors

This test verifies that every field in the Custom Resources' status sections have a corresponding descriptor listed in
the CSV. This test has a maximum score equal to the total number of fields in the status sections of each custom resource passed in via the `--cr-manifest` flag.

## Extending the Scorecard with Plugins

To allow the scorecard to be further extended and capable of more complex testing as well as allow the community to make their own scorecard tests, a plugin system has been implemented
for the scorecard. To use it, a user simply needs to add a binary or script to a `scorecard/bin` directory in their operator's root directory that runs the plugin. It is
recommended to run plugins with a script to allow more configuration from the user and place other required assets in another subdirectory in the `scorecard` directory, ex. `scorecard/assets`.
The scorecard runs all plugins from the root `scorecard` directory. Since the scorecard will run all executable files in the scorecard's
`bin` subdirectory, the plugins can be written in any programming language supported by the OS the scorecard is being run on.

To provide results to the scorecard, the plugin must output a valid JSON object to its `stdout`. Invalid JSON in `stdout` will result in the plugin being marked as failed.
To provide logs to the scorecard, plugins can either set the `log` field for the scorecard suites they return or they can output logs to `stderr`, which will stream the log
to the console if the scorecard is being run in with `--output=human-readable` or be added to the main ScorecardOutput `log` field when being run with `--output=json`.

### JSON format

The JSON output is formatted in the same way that a Kubernetes API would be, which allows for updates to the schema as well as the use of various
Kubernetes helpers. The Golang structs are defined in `pkg/apis/scorecard/v1alpha1/types.go` and can be easily implemented by plugins written in Golang. Below is the JSON Schema:

```json
{
  "$schema": "http://json-schema.org/draft-04/schema#",
  "$ref": "#/definitions/ScorecardOutput",
  "definitions": {
    "ScorecardOutput": {
      "required": [
        "apiVersion",
        "kind",
        "results"
      ],
      "properties": {
        "apiVersion": {
          "type": "string",
          "description": "Version of the object. Ex: osdk.openshift.io/v1alpha1"
        },
        "kind": {
          "type": "string",
          "description": "This should be set to ScorecardOutput"
        },
        "log": {
          "type": "string",
          "description": "Log contains the scorecard's log."
        },
        "results": {
          "items": {
            "$schema": "http://json-schema.org/draft-04/schema#",
            "$ref": "#/definitions/ScorecardSuiteResult"
          },
          "type": "array",
          "description": "Results is an array of ScorecardSuiteResult for each suite of the current scorecard run."
        }
      },
      "additionalProperties": false,
      "type": "object"
    },
    "ScorecardSuiteResult": {
      "required": [
        "name",
        "description",
        "error",
        "pass",
        "partialPass",
        "fail",
        "totalTests",
        "totalScorePercent",
        "tests"
      ],
      "properties": {
        "name": {
          "type": "string",
          "description": "Name is the name of the test suite"
        },
        "description": {
          "type": "string",
          "description": "Description is a description of the test suite"
        },
        "error": {
          "type": "integer",
          "description": "Error is the number of tests that ended in the Error state"
        },
        "pass": {
          "type": "integer",
          "description": "Pass is the number of tests that ended in the Pass state"
        },
        "partialPass": {
          "type": "integer",
          "description": "PartialPass is the number of tests that ended in the PartialPass state"
        },
        "fail": {
          "type": "integer",
          "description": "Fail is the number of tests that ended in the Fail state"
        },
        "totalTests": {
          "type": "integer",
          "description": "TotalTests is the total number of tests run in this suite"
        },
        "totalScorePercent": {
          "type": "integer",
          "description": "TotalScorePercent is the total score of this suite as a percentage"
        },
        "tests": {
          "items": {
            "$schema": "http://json-schema.org/draft-04/schema#",
            "$ref": "#/definitions/ScorecardTestResult"
          },
          "type": "array"
        },
        "log": {
          "type": "string"
        }
      },
      "additionalProperties": false,
      "type": "object"
    },
    "ScorecardTestResult": {
      "required": [
        "state",
        "name",
        "description",
        "earnedPoints",
        "maximumPoints",
        "suggestions",
        "errors"
      ],
      "properties": {
        "state": {
          "type": "string",
          "description": "State is the final state of the test. Valid values are: pass, partial_pass, fail, error"
        },
        "name": {
          "type": "string",
          "description": "Name is the name of the test"
        },
        "description": {
          "type": "string",
          "description": "Description describes what the test does"
        },
        "earnedPoints": {
          "type": "integer",
          "description": "EarnedPoints is how many points the test received after running"
        },
        "maximumPoints": {
          "type": "integer",
          "description": "MaximumPoints is the maximum number of points possible for the test"
        },
        "suggestions": {
          "items": {
            "type": "string"
          },
          "type": "array",
          "description": "Suggestions is a list of suggestions for the user to improve their score (if applicable)"
        },
        "errors": {
          "items": {
            "type": "string"
          },
          "type": "array",
          "description": "Errors is a list of the errors that occured during the test (this can include both fatal and non-fatal errors)"
        }
      },
      "additionalProperties": false,
      "type": "object"
    }
  }
}
```

**NOTE:** The `ScorecardOutput` object is designed the same as a Kubernetes API, and thus also has a full `TypeMeta` and `ObjectMeta`. This means that it contains
various other fields such as `selfLink`, `uid`, and others. At the moment, the only required fields and the only fields that will be checked by the scorecard
are the `kind` and `apiVersion` fields as listed in the above JSONSchema.

Example of a valid JSON output:

```json

{
  "kind": "ScorecardOutput",
  "apiVersion": "osdk.openshift.io/v1alpha1",
  "log": "",
  "results": [
    {
      "name": "Custom Scorecard",
      "description": "Custom operator scorecard tests",
      "error": 0,
      "pass": 1,
      "partialPass": 1,
      "fail": 0,
      "totalTests": 2,
      "totalScorePercent": 71,
      "tests": [
        {
          "state": "partial_pass",
          "name": "Operator Actions Reflected In Status",
          "description": "The operator updates the Custom Resources status when the application state is updated",
          "earnedPoints": 2,
          "maximumPoints": 3,
          "suggestions": [
              "Operator should update status when scaling cluster down"
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
  ]
}
```

**NOTE:** The `ScorecardOutput.Log` field is only intended to be used to log the scorecard's output and the scorecard will ignore that field if a plugin provides it.

## Running the scorecard with a deployed CSV

The scorecard can be run using only a [Cluster Service Version (CSV)][olm-csv], providing a way to test cluster-ready and non-SDK operators.

Running with a CSV alone requires both the `--csv-path=<CSV manifest path>` and `--olm-deployed` flags to be set. The scorecard assumes your CSV and relevant CRD's have been deployed onto the cluster using the OLM when using `--olm-deployed`. This [document][olm-deploy-operator] walks through bundling your CSV and CRD's, deploying the OLM on minikube or [OKD][okd], and deploying your operator. Once these steps have been completed, run the scorecard with both the `--csv-path=<CSV manifest path>` and `--olm-deployed` flags.

A few notes:

- As of now, using the scorecard with a CSV does not permit multiple CR manifests to be set through the CLI/config/CSV annotations. You will have to tear down your operator in the cluster, re-deploy, and re-run the scorecard for each CR being tested. In the future the scorecard will fully support testing multiple CR's without requiring users to teardown/standup each time.
- You can either use `--cr-manifest` or your CSV's [`metadata.annotations['alm-examples']`][olm-csv-alm-examples] to provide CR's to the scorecard, but not both.

[cli-reference]: ../sdk-cli-reference.md#scorecard
[writing-tests]: ./writing-e2e-tests.md
[owned-crds]: https://github.com/operator-framework/operator-lifecycle-manager/blob/master/Documentation/design/building-your-csv.md#owned-crds
[alm-examples]: https://github.com/operator-framework/operator-lifecycle-manager/blob/master/Documentation/design/building-your-csv.md#crd-templates
[viper]: https://github.com/spf13/viper/blob/master/README.md
[olm-csv]:https://github.com/operator-framework/operator-lifecycle-manager/blob/master/Documentation/design/building-your-csv.md
[olm-csv-alm-examples]:https://github.com/operator-framework/operator-lifecycle-manager/blob/master/Documentation/design/building-your-csv.md#crd-templates
[olm]:https://github.com/operator-framework/operator-lifecycle-manager
[olm-deploy-operator]:https://github.com/operator-framework/community-operators/blob/master/docs/testing-operators.md
[okd]:https://www.okd.io/
