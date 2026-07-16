package manifests

import (
	_ "embed"
)

var (
	//go:embed "aggregation.clusterrole.yaml"
	AggregationClusterRoleString string

	//go:embed "aggregation-subrole.clusterole.yaml"
	AggregationSubroleClusterRoleString string

	//go:embed "aggregation.clusterolebinding.yaml"
	AggregationClusterRoleBindingString string

	//go:embed "aggregation.kustomization.patch.yaml"
	AggregationKustomizationPatchString string
)
