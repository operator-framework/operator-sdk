package v1alpha1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	CatalogSourceCRDAPIVersion = GroupName + "/" + GroupVersion
	CatalogSourceKind          = "CatalogSource"
)

// SourceType indicates the type of backing store for a CatalogSource
type SourceType string

const (
	// SourceTypeInternal (deprecated) specifies a CatalogSource of type SourceTypeConfigmap
	SourceTypeInternal SourceType = "internal"

	// SourceTypeConfigmap specifies a CatalogSource that generates a configmap-server registry
	SourceTypeConfigmap SourceType = "configmap"

	// SourceTypeGrpc specifies a CatalogSource that can use an operator registry image to generate a
	// registry-server or connect to a pre-existing registry at an address.
	SourceTypeGrpc SourceType = "grpc"
)

type CatalogSourceSpec struct {
	// SourceType is the type of source
	SourceType SourceType `json:"sourceType"`

	// ConfigMap is the name of the ConfigMap to be used to back a configmap-server registry.
	// Only used when SourceType = SourceTypeConfigmap or SourceTypeInternal.
	// +Optional
	ConfigMap string `json:"configMap,omitempty"`

	// Address is a host that OLM can use to connect to a pre-existing registry.
	// Format: <registry-host or ip>:<port>
	// Only used when SourceType = SourceTypeGrpc.
	// Ignored when the Image field is set.
	// +Optional
	Address string `json:"address,omitempty"`

	// Image is an operator-registry container image to instantiate a registry-server with.
	// Only used when SourceType = SourceTypeGrpc.
	// If present, the address field is ignored.
	// +Optional
	Image string `json:"image,omitempty"`

	// Secrets represent set of secrets that can be used to access the contents of the catalog.
	// It is best to keep this list small, since each will need to be tried for every catalog entry.
	// +Optional
	Secrets []string `json:"secrets,omitempty"`

	// Metadata
	DisplayName string `json:"displayName,omitempty"`
	Description string `json:"description,omitempty"`
	Publisher   string `json:"publisher,omitempty"`
	Icon        Icon   `json:"icon,omitempty"`
}

type RegistryServiceStatus struct {
	Protocol         string      `json:"protocol,omitempty"`
	ServiceName      string      `json:"serviceName,omitempty"`
	ServiceNamespace string      `json:"serviceNamespace,omitempty"`
	Port             string      `json:"port,omitempty"`
	CreatedAt        metav1.Time `json:"createdAt,omitempty"`
}

func (s *RegistryServiceStatus) Address() string {
	return fmt.Sprintf("%s.%s.svc.cluster.local:%s", s.ServiceName, s.ServiceNamespace, s.Port)
}

type CatalogSourceStatus struct {
	ConfigMapResource     *ConfigMapResourceReference `json:"configMapReference,omitempty"`
	RegistryServiceStatus *RegistryServiceStatus      `json:"registryService,omitempty"`
	LastSync              metav1.Time                 `json:"lastSync,omitempty"`
}

type ConfigMapResourceReference struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`

	UID             types.UID `json:"uid,omitempty"`
	ResourceVersion string    `json:"resourceVersion,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient

// CatalogSource is a repository of CSVs, CRDs, and operator packages.
type CatalogSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   CatalogSourceSpec   `json:"spec"`
	Status CatalogSourceStatus `json:"status"`
}

func (c *CatalogSource) Address() string {
	if c.Spec.Address != "" {
		return c.Spec.Address
	}
	return c.Status.RegistryServiceStatus.Address()
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CatalogSourceList is a repository of CSVs, CRDs, and operator packages.
type CatalogSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []CatalogSource `json:"items"`
}
