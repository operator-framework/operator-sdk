---
title: Improve `generate csv` CLI
authors:
  - "@estroz"
reviewers:
  - "@joelanford"
  - "@dmesser"
  - "@robszumski"
approvers:
  - "@joelanford"
  - "@dmesser"
  - "@robszumski"
creation-date: 2019-11-27
last-updated: 2019-11-27
status: implementable
see-also:
  - "doc/user/olm-catalog/generating-a-csv.md"  
---

# Improve `generate csv` CLI

## Release Signoff Checklist

- \[x\] Enhancement is `implementable`
- \[ \] Design details are appropriately documented from clear requirements
- \[ \] Test plan is defined
- \[ \] Graduation criteria for dev preview, tech preview, GA
- \[ \] User-facing documentation is created in [operator-sdk/doc][operator-sdk-doc]

## Summary

The ClusterServiceVersion (CSV) generator's entry point, `operator-sdk generate csv`, currently uses a configuration file that points to several files required to generate a CSV manifest from an Operator project. This config was intended to allow Operators either of high complexity or a non-standard project structure to generate CSV manifests. The config file path can be passed to `generate csv` with `--csv-config=<path>`.

## Motivation

The CSV manifest format is currently in `v1alpha1`, indicating that fields may be added, modified, or removed until stabilization. Perhaps certain SDK files are no longer or new files are required after a manifest format change. This implies that the config format will also change, so it must maintained by SDK authors and updated by SDK users when switching versions.

Another issue crops up when updating existing CSV manifests generated with a different set of Operator manifests. Say I have an existing CSV `my-operator.v0.1.0` that contains legacy Deployment and Role manifests (the project has released `v1.0.0` recently, and the current Deployment and Role will not work with Operator `0.1.0`). However this CSV manifest must be maintained because downstream users are still using version `0.1.0` of the Operator. This scenario requires the operator maintainer to do one of two things:

- Update the CSV config just for this update, such that `operator-path` and `role-paths` point to the legacy Deployment and Role manifest files, respectively. They would then revert those changes once the update is complete.
- Maintain a legacy config for `my-operator.v0.1.0` such that the current config does not need to be updated when updating legacy CSV manifests.

Neither option is desirable. The first option is messy; config files should not have to be modified ad-hoc. The second option implies N configs for N Operator versions in the limit, which is not scalable. Plus each config would have to be modified when upgrading SDK versions if the config format changes, or use old SDK versions to update legacy CSV manifests.

In summary: a config file for CSV manifest generation is too heavy-handed, since it may change often in the future, only configures one `operator-sdk` generator, and adds overhead when migrating to new SDK versions. This file should be replaced by a streamlined CLI approach, which this document details.

## Goals

- I should be able to create/update a CSV manifest from a complex Operator SDK-based project structure.
- I should be able to create/update a CSV manifest from a non-standard Operator project structure.
- I should be able to configure CSV manifest generation for non-Go Operator projects (when this feature is implemented).
- I should not have to rely on on-disk configuration to create/update a CSV manifest.

### Non-Goals

- Reduce the ability to configure the CSV manifest generator.
- Implement CSV manifest generation for non-Go Operator projects.

## Proposal

### Implementation Details/Notes/Constraints

`--inputs` will take a list of patterns, which will be matched against files in the Operator project using [`strings.HasPrefix()`](https://golang.org/pkg/strings/#HasPrefix).
  - The default path for Go projects, if `--inputs` is not set, will be `deploy/`. This emulates current behavior.
  - An error is returned if no matches are found for a pattern in the list.
  - Future non-Go CSV generators can use `--inputs` easily, with new/additional defaults set by individual generators.

### Risks and Mitigations

- Removing `--csv-config` will break existing users that have complex Operator SDK-based or non-standard Operator project structures. A version upgrade guide section is required to migrate these users.

## Design Details

A general CLI option `--inputs` to configure the CSV generator handles either situation described above without changes required in future Operator SDK versions.

The `--inputs` option takes a list of patterns to include in the generation process. For example:

```
$ operator-sdk generate csv --csv-version 0.1.0 --inputs config,deploy/legacy/operator.yaml,deploy/legacy/role.yaml
```

would pass all files in the `config/` dir, `deploy/legacy/operator.yaml`, and `deploy/legacy/role.yaml` to the CSV generator and update the legacy `0.1.0` CSV manifest.

### Test Plan

Unit tests will be implemented for scenarios described above.

##### Removing a deprecated feature

- `--csv-config` and all related code will be removed.

### Upgrade / Downgrade Strategy

- There will be a version upgrade guide section dedicated to transitioning users to the new CLI.

## Alternatives

- An `--exclude` flag to which a list of patterns that should be excluded from CSV generation is passed.
  - This alternative would not easily support non-standard Operator project structures.

[operator-sdk-doc]:  https://sdk.operatorframework.io/
