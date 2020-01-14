// Copyright 2020 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MemcachedRSSpec defines the desired state of MemcachedRS
type MemcachedRSSpec struct {
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	NumNodes int32 `json:"numNodes"`
}

// MemcachedRSStatus defines the observed state of MemcachedRS
type MemcachedRSStatus struct {
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	NodeList []string `json:"nodeList"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MemcachedRS is the Schema for the memcachedrs API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=memcachedrs,scope=Namespaced
// +kubebuilder:storageversion
// +operator-sdk:gen-csv:customresourcedefinitions.displayName="MemcachedRS App"
type MemcachedRS struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MemcachedRSSpec   `json:"spec,omitempty"`
	Status MemcachedRSStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MemcachedRSList contains a list of MemcachedRS
type MemcachedRSList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MemcachedRS `json:"items"`
}
