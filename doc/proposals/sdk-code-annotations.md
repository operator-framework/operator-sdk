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
command-specific token = ":command-name" ;
command-specific prefix = global prefix , command-specific token , { command-specific token } ; (* a set of annotations for a sub-command *)
parent path token = "parent" ;
child path token = ".child" ;
full path = parent path token , { child path token } ;
prefixed path = command-specific prefix , ":" , full path ;
value = ['"'] , "text" , ['"'] ;
annotation = prefixed path , "=" , value ;
```

While all token in a prefix up to command-specific prefixes must be followed, a path token can have command-specific structure. If the destination of data in a `.go` file is a YAML manifest with a list of values interpreted from that code, then a parser for an annotation can create a path including values (and symbols) indexing that list.

An example for [`operator-sdk olm-catalog gen-csv`][sdk_cli_ref_gen_csv], using [`etcd-operator`][etcd_operator_api] API code:

```Go
// PodPolicy defines the policy to create pod for the etcd container.
type PodPolicy struct {
	...
	// +operator-sdk:csv-gen:customresourcedefinitions.specDescriptor.path="pod.resources"
	Resources v1.ResourceRequirements `json:"resources,omitempty"`
	...
}
```

`gen-csv` generates a CSV by pulling data from all over an SDK-generated Operator project. By defining a set of CSV-specific paths with a `:gen-csv` command-specific token, a parser of API code annotations will know that the `customresourcedefinitions` spec field for the CSV generated using a type that has a field of type `PodPolicy` will include a `specDescriptor` list entry with a path value of `pod.resources`. Only the `gen-csv`-specific parser knows that `customresourcedefinitions` is a child of `spec`, a `specDescriptor` is a list field. Other commands will ignore this annotation.

A malleable path element structure allows commands to interpret complex annotations. For an annotation set to be user-friendly, these elements must be kept as simple as possible for the given task. Their parser implementation *must* be accompanied by user-friendly documentation that explains how to create annotations for all supported fields, how the parser will interpret those annotations, and any constraints or requirements on paths or values.

[go_build_tags]:https://golang.org/pkg/go/build/#hdr-Build_Constraints
[k8s_code_gen]:https://blog.openshift.com/kubernetes-deep-dive-code-generation-customresources/
[olm_csv]:https://github.com/operator-framework/operator-lifecycle-manager/blob/master/Documentation/design/building-your-csv.md
[sdk_cli_ref_gen_csv]:https://github.com/operator-framework/operator-sdk/blob/master/doc/sdk-cli-reference.md#gen-csv
[etcd_operator_api]:https://github.com/coreos/etcd-operator/blob/387ece1ca4e9af764c9eb569ff995a21b10ba5ee/pkg/apis/etcd/v1beta2/cluster.go
