---
title: Helm Based Operator Watches
linkTitle: Watches
weight: 20
---

The Watches file contains a list of mappings from custom resources, identified
by it's Group, Version, and Kind, to a Helm chart. The Operator
expects this mapping file in a predefined location: `/opt/helm/watches.yaml`

* **group**:  The group of the Custom Resource that you will be watching.
* **version**:  The version of the Custom Resource that you will be watching.
* **kind**:  The kind of the Custom Resource that you will be watching.
* **chart**: Specifies a chart to be executed. 
* **watchDependentResources**: Allows the helm operator to dynamically watch resources that are created by helm (default: `true`).
* **overrideValues**: Values to be used for overriding Helm chart's defaults. For additional information. 
Please refer to [Using override values and passing environment variables to the Helm chart][override-values].

An example Watches file:

```yaml
---
# Simple example mapping Foo to the Foo chart
- version: v1alpha1
  group: foo.example.com
  kind: Foo
  chart: helm-charts/foo
```

[override-values]: /docs/helm/reference/advanced_features/#passing-environment-variables-to-the-helm-chart
