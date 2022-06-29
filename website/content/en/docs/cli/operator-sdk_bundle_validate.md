---
title: "operator-sdk bundle validate"
---
## operator-sdk bundle validate

Validate an operator bundle

### Synopsis

The 'operator-sdk bundle validate' command can validate both content and format of an operator bundle
image or an operator bundle directory on-disk containing operator metadata and manifests. This command will exit
with an exit code of 1 if any validation errors arise, and 0 if only warnings arise or all validators pass.

A valid bundle is defined by the bundle spec (linked below), therefore the default validator ensures a bundle conforms to
that spec. If you want to ensure that your bundle is valid for an optional superset of requirements such as to those
required to publish your operator on operatorhub.io, then you will need to run one or more supported optional validators.
Set '--list-optional' to list which optional validators are supported, and how they are grouped by label.

More information about operator bundles and metadata:
https://github.com/operator-framework/operator-registry/blob/master/docs/design/operator-bundle.md

NOTE: if validating an image, the image must exist in a remote registry, not just locally.


```
operator-sdk bundle validate [flags]
```

### Examples

```
This example assumes you either have a *pullable* bundle image,
or something similar to the following operator bundle layout present locally:

  $ tree ./bundle
  ./bundle
  ├── manifests
  │   ├── cache.my.domain_memcacheds.yaml
  │   └── memcached-operator.clusterserviceversion.yaml
  └── metadata
      └── annotations.yaml

To validate a local bundle:

  $ operator-sdk bundle validate ./bundle

To build and validate a *pullable* bundle image:

  $ operator-sdk bundle validate <some-registry>/<operator-bundle-name>:<tag>

To list and run optional validators, which are specified by a label selector:

  $ operator-sdk bundle validate --list-optional
  NAME           LABELS                     DESCRIPTION
  operatorhub    name=operatorhub           OperatorHub.io metadata validation.
                 suite=operatorframework

To validate a bundle against the entire suite of validators for Operator Framework, in addition to required bundle validators:

  $ operator-sdk bundle validate ./bundle --select-optional suite=operatorframework

The OperatorHub.io validator in the operatorframework optional suite allows you to validate that your manifests can work with a Kubernetes cluster of a particular version using the k8s-version optional key value:

  $ operator-sdk bundle validate ./bundle --select-optional suite=operatorframework --optional-values=k8s-version=1.22

To validate a bundle against the validator for operatorhub.io specifically, in addition to required bundle validators:

  $ operator-sdk bundle validate ./bundle --select-optional name=operatorhub

This validator allows check the bundle against an specific Kubernetes cluster version using the k8s-version optional key value:

  $ operator-sdk bundle validate ./bundle --select-optional name=operatorhub --optional-values=k8s-version=1.22

[Deprecated] To validate a bundle against the (alpha) validator for Community Operators specifically, in addition to required bundle validators:

  $ operator-sdk bundle validate ./bundle --select-optional name=community --optional-values=index-path=bundle.Dockerfile

To validate a bundle against the validator for Good Practices specifically, in addition to required bundle validators:

  $ operator-sdk bundle validate ./bundle --select-optional name=good-practices

To validate a bundle against the (alpha) validator for Deprecated APIs specifically, in addition to required bundle validators:

  $ operator-sdk bundle validate ./bundle --select-optional name=alpha-deprecated-apis --optional-values=k8s-version=1.22

To validate a bundle against an external validator, in addition to required bundle validators:

  $ operator-sdk bundle validate ./bundle --alpha-select-external /path/to/external-validator[:/path/to/optional-second-validator]

To validate a bundle against the (alpha) validator for Multiple Architectures bundle validation, in addition to required bundle validators:

IMPORTANT: To use this option it is required to have access to pull the images defined on the CSV.

  $ operator-sdk bundle validate ./bundle --select-optional name=multiarch 

NOTE: The --optional-values can be used to inform the container-tools that should be used i.e. "--optional-values=container-tools=docker".
The valid values for the container-tools optional value are [docker, podman, none]. If no value is supplied then the command will default to using docker to inspect the images.
More info: https://github.com/operator-framework/api/blob/master/pkg/validation/internal/multiarch.go

```

### Options

```
      --alpha-select-external string                         Selector to select external validators to run. It should be set to a Unix path list ("/path/to/e1.sh:/path/to/e2")
  -h, --help                                                 help for validate
  -b, --image-builder string                                 Tool to pull and unpack bundle images. Only used when validating a bundle image. One of: [docker, podman, none] (default "docker")
      --list-optional                                        List all optional validators available. When set, no validators will be run
      --optional-values --optional-values=k8s-version=1.22   Inform a []string map of key=values which can be used by the validator. e.g. to check the operator bundle against an Kubernetes version that it is intended to be distributed use --optional-values=k8s-version=1.22 (default [])
  -o, --output string                                        Result format for results. One of: [text, json-alpha1]. Note: output format types containing "alphaX" are subject to change and not covered by guarantees of stable APIs. (default "text")
      --select-optional string                               Label selector to select optional validators to run. Run this command with '--list-optional' to list available optional validators
```

### Options inherited from parent commands

```
      --plugins strings   plugin keys to be used for this subcommand execution
      --verbose           Enable verbose logging
```

### SEE ALSO

* [operator-sdk bundle](../operator-sdk_bundle)	 - Manage operator bundle metadata

