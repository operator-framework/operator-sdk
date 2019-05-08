# SDK Code Annotations

Implementation Owner: @estroz

Status: Draft

[Background](#Background)

[Goals](#Goals)

[Design overview](#Design_overview)

## Background

Code can be annotated with special comment lines that tell a code parser information about document code. Two prominent examples are [go build tags][go_build_tags] and Kubernetes [code generators][k8s_code_gen]. The Operator SDK generates code that, after modification by users, is parsed, transformed, and then represented in an alternate form; SDK-specific code annotations can be used to inform SDK-specific tasks, like generating [Cluster Service Versions][olm_csv] from API's.

## Goals

- Define an annotation system that encapsulates current parsing needs and is extensible for future needs.

## Design Overview

Annotations should follow this EBNF:

```
comment token = ? the character(s) that define a line comment in some language, ex. // in go ? ;
global prefix token = "+operator-sdk" ;
global prefix = comment token , global prefix token ;
use case token = ":use-case" ;
use case prefix = global prefix , use case token , { use case token } ; (* a set of annotations for some use case *)
parent path token = "parent" ;
child path token = ".child" ;
full path = parent path token , { child path token } ;
prefixed path = use case prefix , ":" , full path ;
value = ['"'] , "text" , ['"'] ;
annotation = prefixed path , "=" , value ;
```

The "use case" token is meant to encapsulate the usage for an annotation, ex. `:gen-csv` is used for generating CSV's.

A path token should have use-case-specific structure. The following [`etcd-operator`][etcd_operator_api] API code would be successfully parsed by [`operator-sdk olm-catalog gen-csv`][sdk_cli_ref_gen_csv] in the manner described below:

```Go
type EtcdCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ClusterSpec   `json:"spec"`
	Status            ClusterStatus `json:"status"`
}

type ClusterSpec struct {
	// Pod defines the policy to create pod for the etcd pod.
	//
	// Updating Pod does not take effect on any existing etcd pods.
	Pod *PodPolicy `json:"pod,omitempty"`
}

// PodPolicy defines the policy to create pod for the etcd container.
type PodPolicy struct {
	...
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Resource Requirements"
	Resources v1.ResourceRequirements `json:"resources,omitempty"`
	...
}
```

By defining a set of paths with a `:gen-csv` use case token, an annotation parser "knows" that the `EtcdCluster` spec `ClusterSpec` has a `struct` field of type `PodPolicy` that should be included in the CSV manifest `spec.customresourcedefinitions` field as a `specDescriptor` entry with a `displayName` value of `Resource Requirements`.

For an annotation set to be user-friendly, these elements must be kept as simple as possible for the given task. Their parser implementation *must* be accompanied by documentation that explains how to create annotations for all supported fields, how the parser will interpret those annotations, and any constraints or requirements on paths or values.

[go_build_tags]:https://golang.org/pkg/go/build/#hdr-Build_Constraints
[k8s_code_gen]:https://blog.openshift.com/kubernetes-deep-dive-code-generation-customresources/
[olm_csv]:https://github.com/operator-framework/operator-lifecycle-manager/blob/master/Documentation/design/building-your-csv.md
[sdk_cli_ref_gen_csv]:https://github.com/operator-framework/operator-sdk/blob/master/doc/sdk-cli-reference.md#gen-csv
[etcd_operator_api]:https://github.com/coreos/etcd-operator/blob/387ece1ca4e9af764c9eb569ff995a21b10ba5ee/pkg/apis/etcd/v1beta2/cluster.go
