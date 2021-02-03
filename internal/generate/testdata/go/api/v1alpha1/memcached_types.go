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

	"github.com/operator-framework/operator-sdk/internal/generate/testdata/go/api/shared"
)

// MemcachedSpec defines the desired state of Memcached
type MemcachedSpec struct {
	// Size is the size of the memcached deployment
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	Size int32 `json:"size"`

	// List of Providers
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Providers"
	Providers []Provider `json:"providers,omitempty"`

	// A useful shared type.
	Useful shared.UsefulType `json:",inline"`
}

// Provider represents the container for a single provider
type Provider struct {
	// Foo represents the Foo provider
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Foo Provider"
	Foo *FooProvider `json:"foo,omitempty"`
}

// FooProvider represents integration with Foo
type FooProvider struct {
	// CredentialsSecret is a reference to a secret containing authentication details for the Foo server
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Secret Containing the Credentials",xDescriptors="urn:alm:descriptor:io.kubernetes:Secret"
	//+kubebuilder:validation:Required
	CredentialsSecret *SecretRef `json:"credentialsSecret"`
}

// SecretRef represents a reference to an item within a Secret
type SecretRef struct {
	// Name represents the name of the secret
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Name of the secret",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Name string `json:"name"`

	// Namespace represents the namespace containing the secret
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Namespace containing the secret",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Namespace string `json:"namespace"`

	// Key represents the specific key to reference from the secret
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Key within the secret",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Key string `json:"key,omitempty"`
}

// MemcachedStatus defines the observed state of Memcached
type MemcachedStatus struct {
	// Nodes are the names of the memcached pods
	//+operator-sdk:csv:customresourcedefinitions:type=status
	Nodes []string `json:"nodes"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Memcached is the Schema for the memcacheds API
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=memcacheds,scope=Namespaced
//+kubebuilder:storageversion
//+operator-sdk:csv:customresourcedefinitions:displayName="Memcached App Display Name"
type Memcached struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MemcachedSpec   `json:"spec,omitempty"`
	Status MemcachedStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MemcachedList contains a list of Memcached
type MemcachedList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Memcached `json:"items"`
}
