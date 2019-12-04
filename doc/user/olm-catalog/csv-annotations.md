# Code Annotations for Cluster Service Versions

## Overview

This document describes the semantics of Cluster Service Version (CSV) [code annotations][code-annotations-design] and lists all possible annotations.

## Usage

All annotations have a `+operator-sdk:gen-csv` prefix, denoting that they're parsed while executing [`operator-sdk olm-catalog gen-csv`][sdk-cli-ref].

### Paths

Paths are dot-separated string hierarchies with the above prefix that map to CSV [`spec`][csv-spec] field names.

Example: `+operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Pod Count"`

#### customresourcedefinitions

- `customresourcedefinitions`: child path token
	- `displayName`: quoted string or string literal
	- `resources`: quoted string or string literal, in the format `"kind,version,\"name\""` or `` `kind,version,"name"` ``, where `kind`, `version`, and `name` are fields in each CSV `resources` entry
	- `specDescriptors`, `statusDescriptors`: bool, or child path token
		- `displayName`: quoted string or string literal
		- `x-descriptors`: quoted string or string literal comma-separated list of [`x-descriptor`][csv-x-desc] UI hints.

**NOTES**
- `specDescriptors` and `statusDescriptors` with a value of `true` is required for each field to be included in their respective `customresourcedefinitions` CSV fields. See the examples below.
- `customresourcedefinitions` top-level `kind`, `name`, and `version` fields are parsed from API code.
- All `description` fields are parsed from type declaration and `struct` type field comments.
- `path` is parsed out of a field's JSON tag and merged with parent field path's in dot-hierarchy notation.

### Examples

These examples assume `Memcached`, `MemcachedSpec`, and `MemcachedStatus` are the example projects' kind, spec, and status.

1. Set a display name for a `customresourcedefinitions` kind entry:

```go
// +operator-sdk:gen-csv:customresourcedefinitions.displayName="Memcached App"
type Memcached struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MemcachedSpec   `json:"spec,omitempty"`
	Status MemcachedStatus `json:"status,omitempty"`
}
```

2. Set `displayName`, `path`, `x-descriptors`, and `description` on a field for a `customresourcedefinitions.specDescriptors` entry:

```go
type MemcachedSpec struct {
	// Size is the size of the memcached deployment. <-- This will become Size's specDescriptors.description.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Pod Count"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:podCount,urn:alm:descriptor:io.kubernetes:custom"
	Size int32 `json:"size"` // <-- Size's specDescriptors.path is inferred from this JSON tag.
}
```

3. Let the SDK infer all un-annotated paths on a field for a `customresourcedefinitions.specDescriptors` entry:

```go
type MemcachedSpec struct {
	// Size is the size of the memcached deployment.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Size int32 `json:"size"`
}
```

The SDK uses the `Size` fields' `json` tag name as `path`, `Size` as `displayName`, and field comments as `description`.

The SDK also checks `path` elements against a list of well-known path to x-descriptor string mappings and either uses a match as `x-descriptors`, or does not set `x-descriptors`. Supported mappings:

#### Spec x-descriptors

| PATH | X-DESCRIPTOR |
| --- | --- |
| `size` | `urn:alm:descriptor:com.tectonic.ui:podCount` |
| `podCount` | `urn:alm:descriptor:com.tectonic.ui:podCount` |
| `endpoints` | `urn:alm:descriptor:com.tectonic.ui:endpointList` |
| `endpointList` | `urn:alm:descriptor:com.tectonic.ui:endpointList` |
| `label` | `urn:alm:descriptor:com.tectonic.ui:label` |
| `resources` | `urn:alm:descriptor:com.tectonic.ui:resourceRequirements` |
| `resourceRequirements` | `urn:alm:descriptor:com.tectonic.ui:resourceRequirements` |
| `selector` | `urn:alm:descriptor:com.tectonic.ui:selector:` |
| `namespaceSelector` | `urn:alm:descriptor:com.tectonic.ui:namespaceSelector` |
| none | `urn:alm:descriptor:io.kubernetes:` |
| `booleanSwitch` | `urn:alm:descriptor:com.tectonic.ui:booleanSwitch` |
| `password` | `urn:alm:descriptor:com.tectonic.ui:password` |
| `checkbox` | `urn:alm:descriptor:com.tectonic.ui:checkbox` |
| `imagePullPolicy` | `urn:alm:descriptor:com.tectonic.ui:imagePullPolicy` |
| `updateStrategy` | `urn:alm:descriptor:com.tectonic.ui:updateStrategy` |
| `text` | `urn:alm:descriptor:com.tectonic.ui:text` |
| `number` | `urn:alm:descriptor:com.tectonic.ui:number` |
| `nodeAffinity` | `urn:alm:descriptor:com.tectonic.ui:nodeAffinity` |
| `podAffinity` | `urn:alm:descriptor:com.tectonic.ui:podAffinity` |
| `podAntiAffinity` | `urn:alm:descriptor:com.tectonic.ui:podAntiAffinity` |
| none | `urn:alm:descriptor:com.tectonic.ui:fieldGroup:` |
| none | `urn:alm:descriptor:com.tectonic.ui:arrayFieldGroup:` |
| none | `urn:alm:descriptor:com.tectonic.ui:select:` |
| `advanced` | `urn:alm:descriptor:com.tectonic.ui:advanced` |

#### Status x-descriptors

| PATH | X-DESCRIPTOR |
| --- | --- |
| `podStatuses` | `urn:alm:descriptor:com.tectonic.ui:podStatuses` |
| `size` | `urn:alm:descriptor:com.tectonic.ui:podCount` |
| `podCount` | `urn:alm:descriptor:com.tectonic.ui:podCount` |
| `link` | `urn:alm:descriptor:org.w3:link` |
| `w3link` | `urn:alm:descriptor:org.w3:link` |
| `conditions` | `urn:alm:descriptor:io.kubernetes.conditions` |
| `text` | `urn:alm:descriptor:text` |
| `prometheusEndpoint` | `urn:alm:descriptor:prometheusEndpoint` |
| `phase` | `urn:alm:descriptor:io.kubernetes.phase` |
| `k8sPhase` | `urn:alm:descriptor:io.kubernetes.phase` |
| `reason` | `urn:alm:descriptor:io.kubernetes.phase:reason` |
| `k8sReason` | `urn:alm:descriptor:io.kubernetes.phase:reason` |
| none | `urn:alm:descriptor:io.kubernetes:` |

**NOTE:** any x-descriptor that ends in `:` will not be inferred by `path` element, ex. `urn:alm:descriptor:io.kubernetes:`. Use the `x-descriptors` annotation if you want to enable one for your type.

4. A comprehensive example:
- Infer `path`, `description`, `displayName`, and `x-descriptors` for `specDescriptors` and `statusDescriptors` entries.
- Create three `resources` entries each with `kind`, `version`, and `name` values.

```go
// Represents a cluster of Memcached apps
// +operator-sdk:gen-csv:customresourcedefinitions.displayName="Memcached App"
// +operator-sdk:gen-csv:customresourcedefinitions.resources="Deployment,v1,\"memcached-operator\""
// +operator-sdk:gen-csv:customresourcedefinitions.resources=`Service,v1,"memcached-operator"`
type Memcached struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MemcachedSpec   `json:"spec,omitempty"`
	Status MemcachedStatus `json:"status,omitempty"`
}

type MemcachedSpec struct {
	Pods MemcachedPods `json:"pods"`
}

type MemcachedStatus struct {
	Pods MemcachedPods `json:"podStatuses"`
}

type MemcachedPods struct {
	// Size is the size of the memcached deployment.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Size int32 `json:"size"`
}
```

The generated `customresourcedefinitions` will look like:

```yaml
customresourcedefinitions:
  owned:
  - description: Represents a cluster of Memcached apps
    displayName: Memcached App
    kind: Memcached
    name: memcacheds.cache.example.com
    version: v1alpha1
    resources:
    - kind: Deployment
      name: A Kubernetes Deployment
      version: v1
    - kind: ReplicaSet
      name: A Kubernetes ReplicaSet
      version: v1beta2
    - kind: Pod
      name: A Kubernetes Pod
      version: v1
    specDescriptors:
    - description: The desired number of member Pods for the deployment.
      displayName: Size
      path: pods.size
      x-descriptors:
      - 'urn:alm:descriptor:com.tectonic.ui:podCount'
    statusDescriptors:
    - description: The desired number of member Pods for the deployment.
      displayName: Size
      path: podStatuses.size
      x-descriptors:
      - 'urn:alm:descriptor:com.tectonic.ui:podStatuses'
      - 'urn:alm:descriptor:com.tectonic.ui:podCount'
```

[code-annotations-design]:../../proposals/sdk-code-annotations.md
[sdk-cli-ref]:../../sdk-cli-reference.md#gen-csv
[csv-x-desc]:https://github.com/openshift/console/blob/feabd61/frontend/packages/operator-lifecycle-manager/src/components/descriptors/types.ts#L3-L39
[csv-spec]:https://github.com/operator-framework/operator-lifecycle-manager/blob/e0eea22/doc/design/building-your-csv.md
