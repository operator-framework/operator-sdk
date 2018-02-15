package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type PlayServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []PlayService `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type PlayService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              PlayServiceSpec   `json:"spec"`
	Status            PlayServiceStatus `json:"status,omitempty"`
}

type PlayServiceSpec struct {
	Replica int32 `json:"replica,omitempty"`
	// Fills me
}

type PlayServiceStatus struct {
	// Fills me
}
