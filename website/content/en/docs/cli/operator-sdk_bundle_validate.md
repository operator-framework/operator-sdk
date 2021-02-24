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
  operatorhub    name=operatorhub           OperatorHub.io metadata validation
                 suite=operatorframework

To validate a bundle with a superset of requirements for Operator Framework:
	
  $ operator-sdk bundle validate ./bundle --select-optional suite=operatorframework

To validate a bundle with a superset of requirements for operatorhub.io specifically:
	
  $ operator-sdk bundle validate ./bundle --select-optional name=operatorhub

```

### Options

```
  -h, --help                     help for validate
  -b, --image-builder string     Tool to pull and unpack bundle images. Only used when validating a bundle image. One of: [docker, podman, none] (default "docker")
      --list-optional            List all optional validators available. When set, no validators will be run
  -o, --output string            Result format for results. One of: [text, json-alpha1]. Note: output format types containing "alphaX" are subject to change and not covered by guarantees of stable APIs. (default "text")
      --select-optional string   Label selector to select optional validators to run. Run this command with '--list-optional' to list available optional validators
```

### Options inherited from parent commands

```
      --plugins strings          plugin keys of the plugin to initialize the project with
      --project-version string   project version
      --verbose                  Enable verbose logging
```

### SEE ALSO

* [operator-sdk bundle](../operator-sdk_bundle)	 - Manage operator bundle metadata

