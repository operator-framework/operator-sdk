# Operator Scorecard

The Operator Scorecard is a testing utility included in the `operator-sdk` binary that guides users towards operator best practices
by checking the correctness of their operators and CSVs. While the Scorecard is in an early
stage of development, it will gain more functionality and stabilize over time.

## How It Works

The scorecard works by creating all resources required by CRs and the operator. For the operator
deployment, it also adds another container to the operator's pod that is used to record calls to the API server,
which are analyzed by the scorecard for various tests. The scorecard will also analyze the CR object itself,
modifying spec fields and monitoring how the operator responds.

## Requirements

- An operator made using the `operator-sdk` or an operator that uses a config getter that supports reading from the `KUBECONFIG` environment variable (such as the `clientcmd` or `controller-runtime` config getters). This is required for the scorecard proxy to work correctly.
- Resource manifests for installing/configuring the operator and custom resources (see the [Writing E2E Tests][writing-tests] doc for more information on the global and namespaced manifests).
- (OLM tests only) A CSV file for your operator.

## Running the tests

The scorecard currently uses a large amount of flags to configure the scorecard tests. You can see
these flags in the `scorecard` subcommand help text, or in the [SDK CLI Reference][cli-reference] doc. Here, we will highlight a few important
flags:

- `--cr-manifest` - this is a required flag for the scorecard. This flag must point to the location of the manifest for the custom resource you are currently testing.
- `--csv-path` - this flag is required if the OLM tests are enabled (the tests are enabled by default). This flag must point to the location of the operators' CSV file.
- `--namespaced-manifest` - if set, this flag must point to a manifest file with all resources that run within a namespace. By default, the scorecard will combine `service_account.yaml`, `role.yaml`, `role_binding.yaml`, and `operator.yaml` from the `deploy` directory into a temporary manifest to use as the namespaced manifest.
- `--global-manifest` - if set, this flag must point to all required resources that run globally (not namespaced). By default, the scorecard will combine all CRDs in the `deploy/crds` directory into a temporary manifest to use as the global manifest.
- `--namespace` - if set, which namespace to run the scorecard tests in. If it is not set, the scorecard will use the default namespace of the current context set in the kubeconfig file.

To run the tests, simply run the `scorecard` subcommand from your project root with the flags you want to
use. For example:

```console
$ operator-sdk scorecard --cr-manifest deploy/crds/app_operator_cr.yaml --csv-path deploy/app_operator-0.0.2.yaml
```

## Config File

The scorecard supports the use of a config file instead of or in addition to flags for configuration. By default, the scorecard will look
for a file called `.osdk-scorecard` with either a `.yaml`, `.json`, or `.toml` file extension. You can also
specify a different config file with the `--config` flag. The configuration options in the config file match the flags.
For instance, for the flags `--cr-manifest "deploy/crds/cache_v1alpha1_memcached_cr.yaml" --init-timeout 60 --csv-path "deploy/olm-catalog/memcached-operator/0.0.2/memcached-operator.v0.0.2.clusterserviceversion.yaml"`, the corresponding yaml config file would contain:

```yaml
cr-manifest: "deploy/crds/cache_v1alpha1_memcached_cr.yaml"
init-timeout: 60
csv-path: "deploy/olm-catalog/memcached-operator/0.0.2/memcached-operator.v0.0.2.clusterserviceversion.yaml"
```

The hierarchy of config methods from highest priority to least is: flag->file->default.

The config file support is provided by the `viper` package. For more info on how viper
configuration works, see [`viper`'s README][viper].

## What Each Test Does

There are 8 tests the scorecard can run:

### Basic Operator

#### Spec Block Exists

This test checks the Custom Resource that is created in the cluster to make sure that it has a spec block. This test
has a maximum score of 1.

#### Status Block Exists

This test checks the Custom Resource that is created in the cluster to make sure that it has a status block. This
test has a maximum score of 1.

#### Writing Into CRs Has An Effect

This test reads the scorecard proxy's logs to verify that the operator is making `PUT` and/or `POST` requests to the
API server, indicating that it is modifying resources. This test has a maximum score of 1.

### OLM Integration

#### Provided APIs have validation

This test verifies that all the CRDs in the CRDs folder contain a validation section. If the CRD matches the kind and version of the
CR currently being tested, it will also verify that there is a validation for each spec and status field in that CR. This test has a
maximum score of 1.

#### Owned CRDs Have Resources Listed

This test makes sure that the CRDs listed in the [`owned` CRDs section][owned-crds] of the CSV have a `resources` subsection. This
test has a maximum score equal to the number of CRDs listed in the CSV.

Note: In the future, this test will verify that all resources modified by the operator are listed in the resources section.

#### CRs Have At Least 1 Example

This test checks that the CSV has an [`alm-examples` section][alm-examples] in its metadatas' annotations. This test has a maximum score of 1.

#### Spec Fields With Descriptors

This test verifies that every field in the Custom Resource's spec section has a corresponding descriptor listed in
the CSV. This test has a maximum score equal to the number of fields in the spec section of your Custom Resource.

#### Status Fields With Descriptors

This test verifies that every field in the Custom Resource's status section has a corresponding descriptor listed in
the CSV. This test has a maximum score equal to the number of fields in the status section of your Custom Resource.

[cli-reference]: ../sdk-cli-reference.md#scorecard
[writing-tests]: ./writing-e2e-tests.md
[owned-crds]: https://github.com/operator-framework/operator-lifecycle-manager/blob/master/Documentation/design/building-your-csv.md#owned-crds
[alm-examples]: https://github.com/operator-framework/operator-lifecycle-manager/blob/master/Documentation/design/building-your-csv.md#crd-templates
[viper]: https://github.com/spf13/viper/blob/master/README.md
