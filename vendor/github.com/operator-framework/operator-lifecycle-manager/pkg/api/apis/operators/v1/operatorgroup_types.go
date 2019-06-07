package v1

import (
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	OperatorGroupAnnotationKey             = "olm.operatorGroup"
	OperatorGroupNamespaceAnnotationKey    = "olm.operatorNamespace"
	OperatorGroupTargetsAnnotationKey      = "olm.targetNamespaces"
	OperatorGroupProvidedAPIsAnnotationKey = "olm.providedAPIs"

	OperatorGroupKind = "OperatorGroup"
)

// OperatorGroupSpec is the spec for an OperatorGroup resource.
type OperatorGroupSpec struct {
	// Selector selects the OperatorGroup's target namespaces.
	// +optional
	Selector *metav1.LabelSelector `json:"selector,omitempty"`

	// TargetNamespaces is an explicit set of namespaces to target.
	// If it is set, Selector is ignored.
	// +optional
	TargetNamespaces []string `json:"targetNamespaces,omitempty"`

	// ServiceAccount to bind OperatorGroup roles to.
	ServiceAccount corev1.ServiceAccount `json:"serviceAccount,omitempty"`

	// Static tells OLM not to update the OperatorGroup's providedAPIs annotation
	// +optional
	StaticProvidedAPIs bool `json:"staticProvidedAPIs,omitempty"`
}

// OperatorGroupStatus is the status for an OperatorGroupResource.
type OperatorGroupStatus struct {
	// Namespaces is the set of target namespaces for the OperatorGroup.
	Namespaces []string `json:"namespaces,omitempty"`

	// LastUpdated is a timestamp of the last time the OperatorGroup's status was Updated.
	LastUpdated metav1.Time `json:"lastUpdated"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient

// OperatorGroup is the unit of multitenancy for OLM managed operators.
// It constrains the installation of operators in its namespace to a specified set of target namespaces.
type OperatorGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   OperatorGroupSpec   `json:"spec"`
	Status OperatorGroupStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OperatorGroupList is a list of OperatorGroup resources.
type OperatorGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []OperatorGroup `json:"items"`
}

func (o *OperatorGroup) BuildTargetNamespaces() string {
	sort.Strings(o.Status.Namespaces)
	return strings.Join(o.Status.Namespaces, ",")
}
