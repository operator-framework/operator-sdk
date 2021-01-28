---
title: OpenAPI validation
linkTitle: OpenAPI validation
weight: 70
---

OpenAPIv3 schemas are added to CRD manifests in the `spec.validation` block when the manifests are generated. This validation block allows Kubernetes to validate the properties in a Memcached Custom Resource when it is created or updated.

[Markers][markers] (annotations) are available to configure validations for your API. These markers will always have a `+kubebuilder:validation` prefix.

Usage of markers in API code is discussed in the kubebuilder [CRD generation][generating-crd] and [marker][markers] documentation. A full list of OpenAPIv3 validation markers can be found [here][crd-markers].

To learn more about OpenAPI v3.0 validation schemas in CRDs, refer to the [Kubernetes Documentation][doc-validation-schema].

[markers]: https://book.kubebuilder.io/reference/markers.html
[crd-markers]: https://book.kubebuilder.io/reference/markers/crd-validation.html
[generating-crd]: https://book.kubebuilder.io/reference/generating-crd.html
[doc-validation-schema]: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#specifying-a-structural-schema
