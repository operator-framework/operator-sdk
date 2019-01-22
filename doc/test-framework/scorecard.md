# Operator Scorecard

The Operator Scorecard is a testing utility that is part of the `operator-sdk` binary that allows users to check the
correctness of their operators and CSVs to make sure they follow best practices. It is currently in its very early
stages and will gain more functionality and become more stable over time.

## How It Works

The scorecard works by creating all of the required resources for the CRs and operator. For the operator
deployment, it also adds another container to the operator's pod that is used to intercept calls to the API server,
which can then be analyzed by the scorecard for various tests. The scorecard will also analyze the CR object itself,
modifying spec fields and monitoring how the operator responds.

## Requirements

- Operator made using the `operator-sdk` or using a config getter that supports reading from the `KUBECONFIG` environment variable (such as the `clientcmd` or `controller-runtime` config getters). This is required for the scorecard proxy to work correctly.
- Resource manifests for installing/configuring the operator and custom resources (see the [Writing E2E Tests][writing-tests] doc for more information on the global and namespaced manifests)
- (For OLM tests) A CSV file for your operator

## Running the tests

The scorecard currently has a large amount of flags to allow for configuration of the scorecard tests. You can see
all of the flags in the help text for the command in the [SDK CLI Reference][cli-reference] doc. Here, we will highlight a few important
flags:

- `--cr-manifest` - this is a required flag for the scorecard. This flag must point to the location of the manifest for the custom resource you are currently testing.
- `--csv-path` - this flag is required if the OLM tests are enabled (the tests are enabled by default). This flag must point to the location of the CSV file for this resource.
- `--namespaced-manifest` - if set, this flag must point to a manifest file with all resources that run within a namespace. By default, the scorecard will combine `service_account.yaml`, `role.yaml`, `role_binding.yaml`, and `operator.yaml` from the `deploy` directory into a temporary manifest and use that as the namespaced manifest.
- `--global-manifest` - if set, this flag must point to all required resources that run globally (not namespaced). By default, the scorecard will combine all CRDs in the `deploy/crds` directory into a temporary manifest and use that as the global manifest.
- `--namespace` - if set, which namespace to run the scorecard tests in. If it is not set, the scorecard will use the default namespace of the current context set in the kubeconfig file.

To run the tests, simply run the operator-sdk scorecard subcommand from your project root with the flags you want to
use. For example:

```console
$ operator-sdk scorecard --cr-manifest deploy/crds/app_operator_cr.yaml --csv-path deploy/app_operator-0.0.2.yaml
```

## What Each Test Does

There are currently 8 tests that the scorecard can run:

### Basic Operator

#### Spec Block Exists

This test checks the Custom Resource that is created in the cluster to make sure that it has a spec block. This test
has a maximum score of 1.

#### Status Block Exists

This test checks the Custom Resource that is created in the cluster to make sure that it has a status block. This
test has a maximum score of 1.

#### Operator Action Are Reflected In Status

This test makes modifications to each field in the Custom Resources spec block and then waits and verifies that the
operator updates the status block. This is somewhat prone to breakage as it can potentially change a spec field to an
invalid value. We plan to partially replace this test with user defined tests. This test has a maximum score of 1.

#### Writing Into CRs Has An Effect

This test reads the scorecard proxy's logs to verify that the operator is making `PUT` and/or `POST` requests to the
API server, indicating that it is modifying resources. This test has a maximum score of 1.

### OLM Integration

#### Owned CRDs Have Resources Listed

This test makes sure that the CRDs listed in the Owned CRDs section of the CSV have a resources subsection. In the
future, this test will verify that all resources modified by the operator are listed in the resources section. This
test has a maximum score equal to the number of CRDs listed in the CSV.

#### CRs Have At Least 1 Example

This test checks that the CSV has an `alm-examples` section in its annotations. This test has a maximum score of 1.

#### Spec Fields With Descriptors

This test verifies that every field in the Custom Resource's spec section has a corresponding descriptor listed in
the CSV. This test has a maximum score equal to the number of fields in the spec section of your Custom Resource.

#### Status Fields With Descriptors

This test verifies that every field in the Custom Resource's status section has a corresponding descriptor listed in
the CSV. This test has a maximum score equal to the number of fields in the status section of your Custom Resource.

[cli-reference]: ../sdk-cli-reference.md#scorecard
[writing-tests]: ./writing-e2e-tests.md
