---
title: Dry-run Upgrade Validation in Helm-based Operators
linkTitle: Dry-run Upgrade Validation
weight: 210
description: Skip OpenAPI validation in the dry-run upgrade to reduce the CPU that each reconcile uses.
---

To find out if a release needs an upgrade, each reconcile runs a dry-run upgrade and compares its manifest with the
deployed one. By default, Helm validates the dry-run manifest against the OpenAPI schemas of the cluster. For charts
with many resources, or with resources whose CRDs have large schemas, this validation can use most of the CPU of a
reconcile. An operator that manages many custom resources is then limited by CPU, and a larger
`--max-concurrent-reconciles` does not help.

The `--disable-dry-run-openapi-validation` flag skips the OpenAPI validation in the dry-run upgrade. Installs and
upgrades still validate their manifests, so a changed manifest is validated before it is applied. A manifest that has
not changed since the last install or upgrade is not validated again.

With faster reconciles, more installs and upgrades can run at the same time, up to `--max-concurrent-reconciles`.
Make sure that the operator has enough memory for that many concurrent upgrades, for example after a chart change
that upgrades every release.

For example:

```sh
helm-operator run --disable-dry-run-openapi-validation
```

With the default scaffolding, add the flag to the container args in `config/manager/manager.yaml`. The note about
`config/default/manager_metrics_patch.yaml` in [Maximum Concurrent Reconciles](../max_concurrent_reconciles) applies
to this flag too.
