# operator-sdk scorecard

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
<!-- **Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)* -->

- [Overview](#overview)
- [Requirements](#requirements)
- [Running the Scorecard](#running-the-scorecard)
- [Configuration](#configuration)
  - [Config File](#config-file)
  - [Command Args](#command-args)
  - [Config File Options](#config-file-options)
  - [Plugins](#plugins)
    - [Basic and OLM](#basic-and-olm)
    - [External Plugins](#external-plugins)
- [Tests Performed](#tests-performed)
  - [Basic Operator](#basic-operator)
  - [OLM Integration](#olm-integration)
- [Exit Status](#exit-status)
- [Extending the Scorecard with Plugins](#extending-the-scorecard-with-plugins)
  - [JSON format](#json-format)
- [Running the scorecard with an OLM-managed operator](#running-the-scorecard-with-an-olm-managed-operator)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Overview

The scorecard works by creating all resources required by CRs and the operator.

The scorecard will create another container in the operator deployment which is used to record calls to the API server and run a lot of the tests. The tests performed will also examine some of the fields in the CRs.

The scorecard also supports plugins which allows to extend the functionality of the scorecard and add additional tests on it.

## Requirements

Following are some requirements for the operator project which would be  checked by the scorecard.

- Access to a Kubernetes v1.11.3+ cluster

**For non-SDK operators:**

- Resource manifests for installing/configuring the operator and custom resources. (see the [Writing E2E Tests][writing-tests] doc for more information on the global and namespaced manifests)
- Config getter that supports reading from the `KUBECONFIG` environment variable (such as the `clientcmd` or `controller-runtime` config getters). This is required for the scorecard proxy to work correctly.

**NOTE:** If you would like to use it to check the integration of your operator project with [OLM][olm] then also the [Cluster Service Version (CSV)][olm-csv] file will be required. This is a requirement when the `olm-deployed` option is used

## Running the Scorecard

1. Setup the `.osdk-scorecard.yaml` configuration file in your project.. See [Config file](#config-file)
2. Create the namespace defined in the RBAC files(`role_binding`)
3. Then, run the scorecard, for example `$ operator-sdk scorecard`. See the [Command args](#command-args) to check its options.

**NOTE:** If your operator is non-SDK then some steps will be required in order to meet its requirements.

## Configuration

The scorecard is configured by a config file that allows configuring both internal and external plugins as well as a few global configuration options.

### Config File

To use scorecard, you need to create a config file which by default will be `<project_dir>/.osdk-scorecard.*`. The following is an example of what a YAML formatted config file may look like:

```yaml
scorecard:
  # Setting a global scorecard option
  output: json
  plugins:
    # `basic` tests configured to test 2 CRs
    - basic:
        cr-manifest:
          - "deploy/crds/cache.example.com_v1alpha1_memcached_cr.yaml"
          - "deploy/crds/cache.example.com_v1alpha1_memcachedrs_cr.yaml"
    # `olm` tests configured to test 2 CRs
    - olm:
        cr-manifest:
          - "deploy/crds/cache.example.com_v1alpha1_memcached_cr.yaml"
          - "deploy/crds/cache.example.com_v1alpha1_memcachedrs_cr.yaml"
        csv-path: "deploy/olm-catalog/memcached-operator/0.0.3/memcached-operator.v0.0.3.clusterserviceversion.yaml"
    # Configuring external plugin with no args
    - name: Experimental Plugin
      external:
        command: bin/experiment.sh
      # Configuring an external plugin with args and env
    - name: Custom Test v2
      external:
        command: bin/my-test.sh
        args: ["--version=2"]
        env:
          - name: KUBECONFIG
            value: "~/.kube/config2"
```

The hierarchy of config methods for the global options that are also configurable via a flag from highest priority to least is: flag->file->default.

The config file support is provided by the `viper` package. For more info on how viper configuration works, see [`viper`'s README][viper].

**NOTE:** The config file can be in any of the `json`, `yaml`, or `toml` formats as long as the file has the correct extension. As the config file may be extended to allow configuration
of all `operator-sdk` subcommands in the future, the scorecard's configuration must be under a `scorecard` subsection.

### Command Args

While most configuration is done via a config file, there are a few important args that can be used as follows.

| Flag        | Type   | Description   |
| --------    | -------- | -------- |
| `--bundle`, `-b`  | string |  The path to a bundle directory used for the bundle validation test. |
| `--config`  | string | Path to config file (default `<project_dir>/.osdk-scorecard`; file type and extension can be any of `.yaml`, `.json`, or `.toml`). If a config file is not provided and a config file is not found at the default location, the scorecard will exit with an error. |
| `--output`, `-o`  | string | Output format. Valid options are: `text` and `json`. The default format is `text`, which is designed to be a simpler human readable format. The `json` format uses the JSON schema output format used for plugins defined later in this document. |
| `--kubeconfig`, `-o`  | string |  path to kubeconfig. It sets the kubeconfig internally for internal plugins and sets the `KUBECONFIG` env var to the provided value for external plugins. If an external plugin specifically sets the `KUBECONFIG` env var, the kubeconfig from the specified env var will be used for that plugin instead. |
| `--version`  | string |  The version of scorecard to run, v1alpha2 is the default, valid values are v1alpha and v1alpha2. |
| `--selector`, `-l`  | string |  The label selector to filter tests on, only valid in version v1alpha2. |
| `--list`, `-L`  | bool |  If true, only print the test names that would be run based on selector filtering, only valid in version v1alpha2. |

### Config File Options

| Option        | Type   | Description   |
| --------    | -------- | -------- |
 `bundle` | string | equivalent of the `--bundle` flag. OLM bundle directory path, when specified runs bundle validation |
| `output` | string | equivalent of the `--output` flag. If this option is defined by both the config file and the flag, the flag's value takes priority |
| `kubeconfig` | string | equivalent of the `--kubeconfig` flag. If this option is defined by both the config file and the flag, the flag's value takes priority |
| `plugins` | array | this is an array of [Plugins](#plugins).|

### Plugins

A plugin object is used to configure plugins. The possible values for the plugin object are `basic`, `olm`, or `external`.

Note that each Plugin type has different configuration options and they are named differently in the config. Only one of these fields can be set per plugin.

#### Basic and OLM

The `basic` and `olm` internal plugins have the same configuration fields:

| Option        | Type   | Description   |
| --------    | -------- | -------- |
| `cr-manifest` | [\]string | path(s) for CRs being tested.(required if `olm-deployed` is not set or false) |
| `csv-path` | string | path to CSV for the operator (required for OLM tests or if `olm-deployed` is set to true) |
| `olm-deployed` | bool | indicates that the CSV and relevant CRD's have been deployed onto the cluster by the [Operator Lifecycle Manager (OLM)][olm] |
| `kubeconfig` | string | path to kubeconfig. If both the global `kubeconfig` and this field are set, this field is used for the plugin |
| `namespace` | string | namespace to run the plugins in. If not set, the default specified by the kubeconfig is used |
| `init-timeout` | int | time in seconds until a timeout during initialization of the operator |
| `crds-dir` | string | path to directory containing CRDs that must be deployed to the cluster |
| `namespaced-manifest` | string | manifest file with all resources that run within a namespace. By default, the scorecard will combine `service_account.yaml`, `role.yaml`, `role_binding.yaml`, and `operator.yaml` from the `deploy` directory into a temporary manifest to use as the namespaced manifest |
| `global-manifest` | string | manifest containing required resources that run globally (not namespaced). By default, the scorecard will combine all CRDs in the `crds-dir` directory into a temporary manifest to use as the global manifest |

#### External Plugins

The scorecard allows developers to write their own plugins for the scorecard that can be run via an executable binary or script. For more information on developing external plugins,
please see the [Extending the Scorecard with Plugins](#extending-the-scorecard-with-plugins) section. These are the options available to configure external plugins:

| Option        | Type   | Description   |
| --------    | -------- | -------- |
| `command` | string | (required) path to the plugin binary or script. The path can either be absolute or relative to the operator project's root directory. All external plugins are run from the operator project's root directory |
| `args` | \[\]string | arguments to pass to the plugin |
| `env` | array | `env var` objects, which consist of a `name` and `value` field. If a `KUBECONFIG` env var is declared in this array as well as via the top-level `kubeconfig` option, the `KUBECONFIG` from the env array for the plugin is used |

## Tests Performed

Following the description of each internal [Plugin](#plugins). Note that are 8 internal tests across 2 internal plugins that the scorecard can run. If multiple CRs are specified for a plugin, the test environment is fully cleaned up after each CR so each CR gets a clean testing environment.

Each test has a `short name` that uniquely identifies the test.  This is useful for selecting a specific test or tests to run as follows:
```sh
operator-sdk scorecard -o text --selector=test=checkspectest
operator-sdk scorecard -o text --selector='test in (checkspectest,checkstatustest)'
```

### Basic Operator

| Test        | Description   | Short Name |
| --------    | -------- | -------- |
| Spec Block Exists | This test checks the Custom Resource(s) created in the cluster to make sure that all CRs have a spec block. This test has a maximum score of 1 | checkspectest |
| Status Block Exists | This test checks the Custom Resource(s) created in the cluster to make sure that all CRs have a status block. This test has a maximum score of 1 | checkstatustest |
| Writing Into CRs Has An Effect | This test reads the scorecard proxy's logs to verify that the operator is making `PUT` and/or `POST` requests to the API server, indicating that it is modifying resources. This test has a maximum score of 1 | writingintocrshaseffecttest |

### OLM Integration

| Test        | Description   | Short Name |
| --------    | -------- | -------- |
| OLM Bundle Validation | This test validates the OLM bundle manifests found in the bundle directory as specifed by the bundle flag.  If the bundle contents contain errors, then the test result output will include the validator log as well as error messages from the validation library.  See this [document][olm-bundle] for details on OLM bundles.| bundlevalidationtest |
| Provided APIs have validation |This test verifies that the CRDs for the provided CRs contain a validation section and that there is validation for each spec and status field detected in the CR. This test has a maximum score equal to the number of CRs provided via the `cr-manifest` option. | crdshavevalidationtest |
| Owned CRDs Have Resources Listed | This test makes sure that the CRDs for each CR provided via the `cr-manifest` option have a `resources` subsection in the [`owned` CRDs section][owned-crds] of the CSV. If the test detects used resources that are not listed in the resources section, it will list them in the suggestions at the end of the test. This test has a maximum score equal to the number of CRs provided via the `cr-manifest` option. | crdshaveresourcestest |
| Spec Fields With Descriptors | This test verifies that every field in the Custom Resources' spec sections have a corresponding descriptor listed in the CSV. This test has a maximum score equal to the total number of fields in the spec sections of each custom resource passed in via the `cr-manifest` option. | specdescriptorstest |
| Status Fields With Descriptors | This test verifies that every field in the Custom Resources' status sections have a corresponding descriptor listed in the CSV. This test has a maximum score equal to the total number of fields in the status sections of each custom resource passed in via the `cr-manifest` option. | statusdescriptorstest |

## Exit Status

In version v1alpha2, the scorecard return code is 1 if any of the tests executed did not pass and 0 if all selected tests pass.  Version v1alpha1 returns exit code of 0 regardless if tests fail.

## Extending the Scorecard with Plugins

To allow the scorecard to be further extended and capable of more complex testing as well as allow the community to make their own scorecard tests, a plugin system has been implemented
for the scorecard. To use it, a plugin developer simply needs to provide the binary or script, and the user can then configure the scorecard to use the new plugin. Since the scorecard
can run any executable as a plugin, the plugins can be written in any programming language supported by the OS the scorecard is being run on. All plugins are run from the root of the
operator project.

To provide results to the scorecard, the plugin must output a valid JSON object to its `stdout`. Invalid JSON in `stdout` will result in the plugin being marked as errored.
To provide logs to the scorecard, plugins can either set the `log` field for the scorecard suites they return or they can output logs to `stderr`, which will stream the log
to the console if the scorecard is being run in with `output` unset or set to `text`, or be added to the main `ScorecardOutput.Log` field when `output` is set to `json`

### JSON format

The JSON output is formatted in the same way that a Kubernetes API would be, which allows for updates to the schema as well as the use of various Kubernetes helpers. The Golang structs are defined in `pkg/apis/scorecard/v1alpha2/types.go` and can be easily implemented by plugins written in Golang. Below is the JSON Schema:

```json
{
  "$schema": "http://json-schema.org/draft-04/schema#",
  "$ref": "#/definitions/ScorecardOutput",
  "definitions": {
    "FieldsV1": {
      "additionalProperties": false,
      "type": "object"
    },
    "ManagedFieldsEntry": {
      "properties": {
        "apiVersion": {
          "type": "string"
        },
        "fieldsType": {
          "type": "string"
        },
        "fieldsV1": {
          "$schema": "http://json-schema.org/draft-04/schema#",
          "$ref": "#/definitions/FieldsV1"
        },
        "manager": {
          "type": "string"
        },
        "operation": {
          "type": "string"
        },
        "time": {
          "$ref": "#/definitions/Time"
        }
      },
      "additionalProperties": false,
      "type": "object"
    },
    "ObjectMeta": {
      "properties": {
        "annotations": {
          "patternProperties": {
            ".*": {
              "type": "string"
            }
          },
          "type": "object"
        },
        "clusterName": {
          "type": "string"
        },
        "creationTimestamp": {
          "$schema": "http://json-schema.org/draft-04/schema#",
          "$ref": "#/definitions/Time"
        },
        "deletionGracePeriodSeconds": {
          "type": "integer"
        },
        "deletionTimestamp": {
          "$ref": "#/definitions/Time"
        },
        "finalizers": {
          "items": {
            "type": "string"
          },
          "type": "array"
        },
        "generateName": {
          "type": "string"
        },
        "generation": {
          "type": "integer"
        },
        "labels": {
          "patternProperties": {
            ".*": {
              "type": "string"
            }
          },
          "type": "object"
        },
        "managedFields": {
          "items": {
            "$schema": "http://json-schema.org/draft-04/schema#",
            "$ref": "#/definitions/ManagedFieldsEntry"
          },
          "type": "array"
        },
        "name": {
          "type": "string"
        },
        "namespace": {
          "type": "string"
        },
        "ownerReferences": {
          "items": {
            "$schema": "http://json-schema.org/draft-04/schema#",
            "$ref": "#/definitions/OwnerReference"
          },
          "type": "array"
        },
        "resourceVersion": {
          "type": "string"
        },
        "selfLink": {
          "type": "string"
        },
        "uid": {
          "type": "string"
        }
      },
      "additionalProperties": false,
      "type": "object"
    },
    "OwnerReference": {
      "required": [
        "apiVersion",
        "kind",
        "name",
        "uid"
      ],
      "properties": {
        "apiVersion": {
          "type": "string"
        },
        "blockOwnerDeletion": {
          "type": "boolean"
        },
        "controller": {
          "type": "boolean"
        },
        "kind": {
          "type": "string"
        },
        "name": {
          "type": "string"
        },
        "uid": {
          "type": "string"
        }
      },
      "additionalProperties": false,
      "type": "object"
    },
    "ScorecardOutput": {
      "required": [
        "TypeMeta",
        "log",
        "results"
      ],
      "properties": {
        "TypeMeta": {
          "$schema": "http://json-schema.org/draft-04/schema#",
          "$ref": "#/definitions/TypeMeta"
        },
        "log": {
          "type": "string"
        },
        "metadata": {
          "$schema": "http://json-schema.org/draft-04/schema#",
          "$ref": "#/definitions/ObjectMeta"
        },
        "results": {
          "items": {
            "$schema": "http://json-schema.org/draft-04/schema#",
            "$ref": "#/definitions/ScorecardTestResult"
          },
          "type": "array"
        }
      },
      "additionalProperties": false,
      "type": "object"
    },
    "ScorecardTestResult": {
      "required": [
        "name",
        "description"
      ],
      "properties": {
        "description": {
          "type": "string"
        },
        "errors": {
          "items": {
            "type": "string"
          },
          "type": "array"
        },
        "labels": {
          "patternProperties": {
            ".*": {
              "type": "string"
            }
          },
          "type": "object"
        },
        "log": {
          "type": "string"
        },
        "name": {
          "type": "string"
        },
        "state": {
          "type": "string"
        },
        "suggestions": {
          "items": {
            "type": "string"
          },
          "type": "array"
        }
      },
      "additionalProperties": false,
      "type": "object"
    },
    "Time": {
      "additionalProperties": false,
      "type": "object"
    },
    "TypeMeta": {
      "properties": {
        "apiVersion": {
          "type": "string"
        },
        "kind": {
          "type": "string"
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
To add logs to the main `ScorecardOuput.Log` field, a plugin can output the logs to `stderr`.

## Running the scorecard with an OLM-managed operator

The scorecard can be run using a [Cluster Service Version (CSV)][olm-csv], providing a way to test cluster-ready and non-SDK operators.

Running with a CSV alone requires both the `csv-path: <CSV manifest path>` and `olm-deployed` options to be set. The scorecard assumes your CSV and relevant CRD's have been deployed onto the cluster using OLM when using `olm-deployed`.

The scorecard requires a proxy container in the operator's `Deployment` pod to read operator logs. A few modifications to your CSV and creation of one extra object are required to run the proxy _before_ deploying your operator with OLM:

1. Create a proxy server secret containing a local Kubeconfig:
    1. Generate a username using the scorecard proxy's namespaced owner reference.
          ```sh
          # Substitute "$your_namespace" for the namespace your operator will be deployed in (if any).
          $ echo '{"apiVersion":"","kind":"","name":"scorecard","uid":"","Namespace":"'${your_namespace}'"}' | base64 -w 0
          eyJhcGlWZXJzaW9uIjoiIiwia2luZCI6IiIsIm5hbWUiOiJzY29yZWNhcmQiLCJ1aWQiOiIiLCJOYW1lc3BhY2UiOiJvbG0ifQo=
          ```
    1. Write a `Config` manifest `scorecard-config.yaml` using the following template, substituting `${your_username}` for the base64 username generated above:
          ```yaml
          apiVersion: v1
          kind: Config
          clusters:
          - cluster:
              insecure-skip-tls-verify: true
              server: http://${your_username}@localhost:8889
            name: proxy-server
          contexts:
          - context:
              cluster: proxy-server
              user: admin/proxy-server
            name: $namespace/proxy-server
          current-context: $namespace/proxy-server
          preferences: {}
          users:
          - name: admin/proxy-server
            user:
              username: ${your_username}
              password: unused
          ```
    1. Encode the `Config` as base64:
          ```sh
          $ cat scorecard-config.yaml | base64 -w 0
          YXBpVmVyc2lvbjogdjEKa2luZDogQ29uZmlnCmNsdXN0ZXJzOgotIGNsdXN0ZXI6CiAgICBpbnNlY3VyZS1za2lwLXRscy12ZXJpZnk6IHRydWUKICAgIHNlcnZlcjogaHR0cDovL2V5SmhjR2xXWlhKemFXOXVJam9pSWl3aWEybHVaQ0k2SWlJc0ltNWhiV1VpT2lKelkyOXlaV05oY21RaUxDSjFhV1FpT2lJaUxDSk9ZVzFsYzNCaFkyVWlPaUp2YkcwaWZRbz1AbG9jYWxob3N0Ojg4ODkKICBuYW1lOiBwcm94eS1zZXJ2ZXIKY29udGV4dHM6Ci0gY29udGV4dDoKICAgIGNsdXN0ZXI6IHByb3h5LXNlcnZlcgogICAgdXNlcjogYWRtaW4vcHJveHktc2VydmVyCiAgbmFtZTogL3Byb3h5LXNlcnZlcgpjdXJyZW50LWNvbnRleHQ6IC9wcm94eS1zZXJ2ZXIKcHJlZmVyZW5jZXM6IHt9CnVzZXJzOgotIG5hbWU6IGFkbWluL3Byb3h5LXNlcnZlcgogIHVzZXI6CiAgICB1c2VybmFtZTogZXlKaGNHbFdaWEp6YVc5dUlqb2lJaXdpYTJsdVpDSTZJaUlzSW01aGJXVWlPaUp6WTI5eVpXTmhjbVFpTENKMWFXUWlPaUlpTENKT1lXMWxjM0JoWTJVaU9pSnZiRzBpZlFvPQogICAgcGFzc3dvcmQ6IHVudXNlZAo=
          ```
    1. Create a `Secret` manifest `scorecard-secret.yaml` containing the operator's namespace (if any) the `Config`'s base64 encoding as a `spec.data` value under the key `kubeconfig`:
          ```yaml
          apiVersion: v1
          kind: Secret
          metadata:
            name: scorecard-kubeconfig
            namespace: ${your_namespace}
          data:
            kubeconfig: ${kubeconfig_base64}
          ```
    1. Apply the secret in-cluster:
          ```sh
          $ kubectl apply -f scorecard-secret.yaml
          ```
    1. Insert a volume referring to the `Secret` into the operator's `Deployment`:
          ```yaml
          spec:
            install:
              spec:
                deployments:
                - name: memcached-operator
                  spec:
                    ...
                    template:
                      ...
                      spec:
                        containers:
                        ...
                        volumes:
                        # scorecard kubeconfig volume
                        - name: scorecard-kubeconfig
                          secret:
                            secretName: scorecard-kubeconfig
                            items:
                            - key: kubeconfig
                              path: config
          ```
1. Insert a volume mount and `KUBECONFIG` environment variable into each container in your operator's `Deployment`:
    ```yaml
    spec:
      install:
        spec:
          deployments:
          - name: memcached-operator
            spec:
              ...
              template:
                ...
                spec:
                  containers:
                  - name: container1
                    ...
                    volumeMounts:
                    # scorecard kubeconfig volume mount
                    - name: scorecard-kubeconfig
                      mountPath: /scorecard-secret
                    env:
                      # scorecard kubeconfig env
                    - name: KUBECONFIG
                      value: /scorecard-secret/config
                  - name: container2
                    # Do the same for this and all other containers.
                    ...
    ```
1. Insert the scorecard proxy container into the operator's `Deployment`:
    ```yaml
    spec:
      install:
        spec:
          deployments:
          - name: memcached-operator
            spec:
              ...
              template:
                ...
                spec:
                  containers:
                  ...
                  # scorecard proxy container
                  - name: scorecard-proxy
                    command:
                    - scorecard-proxy
                    env:
                    - name: WATCH_NAMESPACE
                      valueFrom:
                        fieldRef:
                          apiVersion: v1
                          fieldPath: metadata.namespace
                    image: quay.io/operator-framework/scorecard-proxy:master
                    imagePullPolicy: Always
                    ports:
                    - name: proxy
                      containerPort: 8889
    ```

Alternatively, the [community-operators][community-operators] repo has several bash functions that can perform these operations for you:
```sh
$ curl -Lo csv-manifest-modifiers.sh https://raw.githubusercontent.com/operator-framework/community-operators/master/scripts/lib/file
$ . ./csv-manifest-modifiers.sh
# $NAMESPACE is the namespace your operator will deploy in
$ create_kubeconfig_secret_file scorecard-secret.yaml "$NAMESPACE"
$ kubectl apply -f scorecard-secret.yaml
# $CSV_FILE is the path to your operator's CSV manifest
$ insert_kubeconfig_volume "$CSV_FILE"
$ insert_kubeconfig_secret_mount "$CSV_FILE"
$ insert_proxy_container "$CSV_FILE" "quay.io/operator-framework/scorecard-proxy:master"
```

Once done, follow the steps in this [document][olm-deploy-operator] to bundle your CSV and CRD's, deploy OLM on minikube or [OKD][okd], and deploy your operator. Once these steps have been completed, run the scorecard with both the `csv-path: <CSV manifest path>` and `olm-deployed` options set.

**NOTES:**

- As of now, using the scorecard with a CSV does not permit multiple CR manifests to be set through the CLI/config/CSV annotations. You will have to tear down your operator in the cluster, re-deploy, and re-run the scorecard for each CR being tested. In the future the scorecard will fully support testing multiple CR's without requiring users to teardown/standup each time.
- You can either set `cr-manifest` or your CSV's [`metadata.annotations['alm-examples']`][olm-csv-alm-examples] to provide CR's to the scorecard, but not both.

[cli-reference]: ../sdk-cli-reference.md#scorecard
[writing-tests]: ./writing-e2e-tests.md
[owned-crds]: https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/design/building-your-csv.md#owned-crds
[alm-examples]: https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/design/building-your-csv.md#crd-templates
[viper]: https://github.com/spf13/viper/blob/master/README.md
[olm-bundle]:https://github.com/operator-framework/operator-registry#manifest-format
[olm-csv]:https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/design/building-your-csv.md
[olm-csv-alm-examples]:https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/design/building-your-csv.md#crd-templates
[olm]:https://github.com/operator-framework/operator-lifecycle-manager
[olm-deploy-operator]:https://github.com/operator-framework/community-operators/blob/master/docs/testing-operators.md
[okd]:https://www.okd.io/
[community-operators]:https://github.com/operator-framework/community-operators
