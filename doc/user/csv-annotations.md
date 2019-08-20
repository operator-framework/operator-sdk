# Code Annotations for Cluster Service Versions

## Overview

This document describes the semantics of Cluster Service Version (CSV) [code annotations][code_annotations_design] and lists all possible annotations.

## Usage

All annotations have a `+operator-sdk:gen-csv:` prefix, denoting that they're parsed while executing [`operator-sdk olm-catalog gen-csv`][sdk_cli_ref].

### Paths

Paths are dot-separated string hierarchies with the above prefix that map to CSV [`spec`][csv_spec] field names.

Example: `+operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Pod Count"`

#### customresourcedefinitions

- `customresourcedefinitions`: child path token
	-	`displayName`: quoted string
	- `resources`: quoted string, in the format `"kind,version,\"name\""`, where `kind`, `version`, and `name` are fields in each CSV `resources` entry
	- `specDescriptors`, `statusDescriptors`: bool, or child path token
		- `displayName`: quoted string
		- `x-descriptors`: quoted string comma-separated list of [`x-descriptor`][csv_x_desc] UI hints.

Notes:
- `specDescriptors` and `statusDescriptors` with a value of `true` is required for each field to be included in their respective `customresourcedefinitions` CSV fields. See the examples below.
- `customresourcedefinitions` top-level `kind`, `name`, and `version` fields are parsed from API code.
- All `description`'s are parsed from type declaration and `struct` type field comments.
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
The SDK also checks `path` elements against a list of well-known path to x-descriptor string [mappings][csv_x_desc_mappings] and either uses a match as `x-descriptors`, or does not set `x-descriptors`.

4. A comprehensive example:
- Infer `path`, `description`, `displayName`, and `x-descriptors` for `specDescriptors` and `statusDescriptors` entries.
- Create three `resources` entries each with `kind`, `version`, and `name` values.

```go
// Represents a cluster of Memcached apps
// +operator-sdk:gen-csv:customresourcedefinitions.displayName="Memcached App"
// +operator-sdk:gen-csv:customresourcedefinitions.resources="Deployment,v1,\"A Kubernetes Deployment\""
// +operator-sdk:gen-csv:customresourcedefinitions.resources="ReplicaSet,v1beta2,\"A Kubernetes ReplicaSet\""
// +operator-sdk:gen-csv:customresourcedefinitions.resources="Pod,v1,\"A Kubernetes Pod\""
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

[code_annotations_design]:../proposals/sdk-code-annotations.md
[sdk_cli_ref]:../sdk-cli-reference.md#gen-csv
[csv_x_desc]:https://github.com/openshift/console/blob/master/frontend/public/components/operator-lifecycle-manager/descriptors/types.ts#L5-L27
[csv_spec]:https://github.com/operator-framework/operator-lifecycle-manager/blob/master/Documentation/design/building-your-csv.md
[csv_x_desc_mappings]:https://github.com/estroz/operator-sdk/blob/8750e06060c3d545097846c6bbfaa55e58654fb5/internal/pkg/descriptor/parse.go#L362-L396
