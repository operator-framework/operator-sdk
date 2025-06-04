---
title: Define Watches in Helm-based operators
linkTitle: Define Watches
weight: 200
description: Specification for the `watches.yaml` file in Helm-based operators.
---

The Watches file contains a list of mappings from custom resources, identified
by it's Group, Version, and Kind, to a Helm chart. The Operator
expects this mapping file in a predefined location: `/opt/helm/watches.yaml`

The follow tables describes the fields in an entry in `watches.yaml`:

| Field                   | Description |
| :---------------------- | :---------- |
| group                   | The group of the Custom Resource that you will be watching. |
| version                 | The version of the Custom Resource that you will be watching. |
| kind                    | The kind of the Custom Resource that you will be watching. |
| chart                   | The path to the helm chart to use when reconciling this GVK.  |
| watchDependentResources | Enable watching resources that are created by helm (default: `true`). |
| overrideValues          | Values to be used for overriding Helm chart's defaults. For additional information see the [reference doc][override-values]. |
| selector                | The conditions that a resource's labels must satisfy in order to get reconciled. For additional information see [labels and selectors documentation][label-selector-doc]. |
| dryRunOption            | The helm dry-run method to use when comparing manifests. Set to `server` to ensure `lookup()` functions are evaluated (default: `client/none`) |


For reference, here is an example of a simple `watches.yaml` file:

```yaml
# Use the 'create api' subcommand to add watches to this file.
- group: foo.example.com
  version: v1alpha1
  kind: Foo
  chart: helm-charts/foo
  overrideValues:
    image.repository: quay.io/mycustomrepo
  watchDependentResources: false
  selector:
    matchExpressions:
     - {key: testLabel, operator: Exists, values: []}
  dryRunOption: server
```

[override-values]: /docs/building-operators/helm/reference/advanced_features/override_values/
[label-selector-doc]: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
