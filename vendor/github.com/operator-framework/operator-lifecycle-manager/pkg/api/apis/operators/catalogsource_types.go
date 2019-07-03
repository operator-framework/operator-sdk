package operators

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// CatalogSourceKind is the PascalCase name of a CatalogSource's kind.
const CatalogSourceKind = "CatalogSource"

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
	SourceType SourceType

	// ConfigMap is the name of the ConfigMap to be used to back a configmap-server registry.
	// Only used when SourceType = SourceTypeConfigmap or SourceTypeInternal.
	// +Optional
	ConfigMap string

	// Address is a host that OLM can use to connect to a pre-existing registry.
	// Format: <registry-host or ip>:<port>
	// Only used when SourceType = SourceTypeGrpc.
	// Ignored when the Image field is set.
	// +Optional
	Address string

	// Image is an operator-registry container image to instantiate a registry-server with.
	// Only used when SourceType = SourceTypeGrpc.
	// If present, the address field is ignored.
	// +Optional
	Image string

	// Secrets represent set of secrets that can be used to access the contents of the catalog.
	// It is best to keep this list small, since each will need to be tried for every catalog entry.
	// +Optional
	Secrets []string

	// Metadata
	DisplayName string
	Description string
	Publisher   string
	Icon        Icon
}

type RegistryServiceStatus struct {
	Protocol         string
	ServiceName      string
	ServiceNamespace string
	Port             string
	CreatedAt        metav1.Time
}

func (s *RegistryServiceStatus) Address() string {
	return fmt.Sprintf("%s.%s.svc.cluster.local:%s", s.ServiceName, s.ServiceNamespace, s.Port)
}

type CatalogSourceStatus struct {
	ConfigMapResource     *ConfigMapResourceReference
	RegistryServiceStatus *RegistryServiceStatus
	LastSync              metav1.Time
}

type ConfigMapResourceReference struct {
	Name      string
	Namespace string

	UID             types.UID
	ResourceVersion string
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient

// CatalogSource is a repository of CSVs, CRDs, and operator packages.
type CatalogSource struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec   CatalogSourceSpec
	Status CatalogSourceStatus
}

func (c *CatalogSource) Address() string {
	if c.Spec.Address != "" {
		return c.Spec.Address
	}
	return c.Status.RegistryServiceStatus.Address()
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CatalogSourceList is a list of CatalogSource resources.
type CatalogSourceList struct {
	metav1.TypeMeta
	metav1.ListMeta

	Items []CatalogSource
}
