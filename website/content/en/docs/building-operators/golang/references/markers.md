---
title: API Markers
linkTitle: API Markers
weight: 40
---

This document describes [code markers][markers] supported by the SDK.

## ClusterServiceVersion markers

This section details ClusterServiceVersion (CSV) code markers and lists available markers.

**Note:** CSV markers can only be used in Go Operator projects. Annotations for Ansible and Helm Operator projects will be added in the future.

### Usage

All CSV markers have the prefix `+operator-sdk:csv`.

#### `+operator-sdk:csv:customresourcedefinitions`

These markers populate [owned `customresourcedefinitions`][csv-crds] in your CSV.

Possible type-level markers:
- `+operator-sdk:csv:customresourcedefinitions:displayName="some display name"`
	- Configures the kind's display name.
- `+operator-sdk:csv:customresourcedefinitions:resources={{Kind1,v1alpha1,"dns-name-1"},{Kind2,v1,"dns-name-2"},...}`
	- Configures the kind's resources.

Possible field-level markers, all of which must contain the `type=[spec,status]` key-value pair:
- `+operator-sdk:csv:customresourcedefinitions:type=[spec,status],displayName="some field display name"`
	- Configures the field's display name.
- `+operator-sdk:csv:customresourcedefinitions:type=[spec,status],xDescriptors="urn:alm:descriptor:com.tectonic.ui:podCount,urn:alm:descriptor:io.kubernetes:custom"`
	- Configures the field's x-descriptors.


Top-level `kind`, `name`, and `version` fields are parsed from API code.
All `description` fields are parsed from type declaration and `struct` type field comments.
All `path` fields are parsed from a field's JSON tag and merged with parent
field path's in dot-hierarchy notation.

##### x-descriptors

Check out the [descriptor reference][csv-x-desc] for available `x-descriptors` paths.

#### Examples

These examples assume `Memcached`, `MemcachedSpec`, and `MemcachedStatus` are the example projects' kind, spec, and status.

1. Set a `displayName` and `resources` for a `customresourcedefinitions` kind entry:

	```go
	// +operator-sdk:csv:customresourcedefinitions:displayName="Memcached App",resources={{Pod,v1,memcached-runner},{Deployment,v1,memcached-deployment}}
	type Memcached struct {
		metav1.TypeMeta   `json:",inline"`
		metav1.ObjectMeta `json:"metadata,omitempty"`

		Spec   MemcachedSpec   `json:"spec,omitempty"`
		Status MemcachedStatus `json:"status,omitempty"`
	}
	```

2. Set `displayName`, `path`, `xDescriptors`, and `description` on a field for a `customresourcedefinitions.specDescriptors` entry:

	```go
	type MemcachedSpec struct {
		// Size is the size of the memcached deployment. <-- This will become Size's specDescriptors.description.
		// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Number of pods",xDescriptors="urn:alm:descriptor:com.tectonic.ui:podCount,urn:alm:descriptor:io.kubernetes:custom"
		Size int32 `json:"size"` // <-- Size's specDescriptors.path is inferred from this JSON tag.
	}
	```

3. Let the SDK infer all unmarked paths on a field for a `customresourcedefinitions.specDescriptors` entry:

	```go
	type MemcachedSpec struct {
		// Size is the size of the memcached deployment.
		// +operator-sdk:csv:customresourcedefinitions:type=spec
		Size int32 `json:"size"`
	}
	```

	The SDK uses the `Size` fields' `json` tag name as `path`, `Size` as `displayName`, and field comments as `description`.

4. A comprehensive example:
	- Infer `path`, `description`, `displayName`, and `x-descriptors` for `specDescriptors` and `statusDescriptors` entries.
	- Create three `resources` entries each with `kind`, `version`, and `name` values.

	```go
	// Represents a cluster of Memcached apps
	// +operator-sdk:csv:customresourcedefinitions:displayName="Memcached App",resources={{Pod,v1,memcached-runner},{Deployment,v1,memcached-deployment}}
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
		Pods 		 MemcachedPods `json:"podStatuses"`
		// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="Pod Count",xDescriptors="urn:alm:descriptor:com.tectonic.ui:podCount"
		PodCount int 					 `json:"podCount"`
	}

	type MemcachedPods struct {
		// Size is the size of the memcached deployment.
		// +operator-sdk:csv:customresourcedefinitions:type=spec
		// +operator-sdk:csv:customresourcedefinitions.type=status
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
	      name: memcached-deployment
	      version: v1
	    - kind: Pod
	      name: memcached-runner
	      version: v1
	    specDescriptors:
	    - description: The desired number of member Pods for the deployment.
	      displayName: Size
	      path: pods.size
	    statusDescriptors:
	    - description: The desired number of member Pods for the deployment.
	      displayName: Size
	      path: podStatuses.size
	    - displayName: Size
	      path: podCount
	      x-descriptors:
	      - 'urn:alm:descriptor:com.tectonic.ui:podCount'
	```


## Deprecated markers

[Markers][deprecated-markers] supported by `operator-sdk` prior to v1.0.0 are deprecated.
You can migrate to the new marker system by running the following script:

```console
$ curl -sSLo migrate-markers.sh https://raw.githubusercontent.com/operator-framework/operator-sdk/master/hack/generate/migrate-markers.sh
$ chmod +x ./migrate-markers.sh
$ ./migrate-markers.sh path/to/*_types.go
```


[markers]:https://pkg.go.dev/sigs.k8s.io/controller-tools/pkg/markers
[cli-gen-kustomize-manifests]:/docs/cli/operator-sdk_generate_kustomize_manifests
[csv-x-desc]:https://github.com/openshift/console/blob/master/frontend/packages/operator-lifecycle-manager/src/components/descriptors/reference/reference.md
[csv-spec]:https://github.com/operator-framework/operator-lifecycle-manager/blob/e0eea22/doc/design/building-your-csv.md
[csv-crds]:https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/design/building-your-csv.md#your-custom-resource-definitions
[deprecated-markers]:https://v0-19-x.sdk.operatorframework.io/docs/golang/references/markers/
