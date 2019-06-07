package operators

import (
	"encoding/json"
	"fmt"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/version"
)

// ClusterServiceVersionKind is the PascalCase name of a CSV's kind.
const ClusterServiceVersionKind = "ClusterServiceVersionKind"

// InstallModeType is a supported type of install mode for CSV installation
type InstallModeType string

const (
	// InstallModeTypeOwnNamespace indicates that the operator can be a member of an `OperatorGroup` that selects its own namespace.
	InstallModeTypeOwnNamespace InstallModeType = "OwnNamespace"
	// InstallModeTypeSingleNamespace indicates that the operator can be a member of an `OperatorGroup` that selects one namespace.
	InstallModeTypeSingleNamespace InstallModeType = "SingleNamespace"
	// InstallModeTypeMultiNamespace indicates that the operator can be a member of an `OperatorGroup` that selects more than one namespace.
	InstallModeTypeMultiNamespace InstallModeType = "MultiNamespace"
	// InstallModeTypeAllNamespaces indicates that the operator can be a member of an `OperatorGroup` that selects all namespaces (target namespace set is the empty string "").
	InstallModeTypeAllNamespaces InstallModeType = "AllNamespaces"
)

// InstallMode associates an InstallModeType with a flag representing if the CSV supports it
type InstallMode struct {
	Type      InstallModeType
	Supported bool
}

// InstallModeSet is a mapping of unique InstallModeTypes to whether they are supported.
type InstallModeSet map[InstallModeType]bool

// NamedInstallStrategy represents the block of an ClusterServiceVersion resource
// where the install strategy is specified.
type NamedInstallStrategy struct {
	StrategyName    string
	StrategySpecRaw json.RawMessage
}

// StatusDescriptor describes a field in a status block of a CRD so that OLM can consume it
type StatusDescriptor struct {
	Path         string
	DisplayName  string
	Description  string
	XDescriptors []string
	Value        *json.RawMessage
}

// SpecDescriptor describes a field in a spec block of a CRD so that OLM can consume it
type SpecDescriptor struct {
	Path         string
	DisplayName  string
	Description  string
	XDescriptors []string
	Value        *json.RawMessage
}

// ActionDescriptor describes a declarative action that can be performed on a custom resource instance
type ActionDescriptor struct {
	Path         string
	DisplayName  string
	Description  string
	XDescriptors []string
	Value        *json.RawMessage
}

// CRDDescription provides details to OLM about the CRDs
type CRDDescription struct {
	Name              string
	Version           string
	Kind              string
	DisplayName       string
	Description       string
	Resources         []APIResourceReference
	StatusDescriptors []StatusDescriptor
	SpecDescriptors   []SpecDescriptor
	ActionDescriptor  []ActionDescriptor
}

// APIServiceDescription provides details to OLM about apis provided via aggregation
type APIServiceDescription struct {
	Name              string
	Group             string
	Version           string
	Kind              string
	DeploymentName    string
	ContainerPort     int32
	DisplayName       string
	Description       string
	Resources         []APIResourceReference
	StatusDescriptors []StatusDescriptor
	SpecDescriptors   []SpecDescriptor
	ActionDescriptor  []ActionDescriptor
}

// APIResourceReference is a Kubernetes resource type used by a custom resource
type APIResourceReference struct {
	Name    string
	Kind    string
	Version string
}

// GetName returns the name of an APIService as derived from its group and version.
func (d APIServiceDescription) GetName() string {
	return fmt.Sprintf("%s.%s", d.Version, d.Group)
}

// CustomResourceDefinitions declares all of the CRDs managed or required by
// an operator being ran by ClusterServiceVersion.
//
// If the CRD is present in the Owned list, it is implicitly required.
type CustomResourceDefinitions struct {
	Owned    []CRDDescription
	Required []CRDDescription
}

// APIServiceDefinitions declares all of the extension apis managed or required by
// an operator being ran by ClusterServiceVersion.
type APIServiceDefinitions struct {
	Owned    []APIServiceDescription
	Required []APIServiceDescription
}

// ClusterServiceVersionSpec declarations tell OLM how to install an operator
// that can manage apps for a given version.
type ClusterServiceVersionSpec struct {
	InstallStrategy           NamedInstallStrategy
	Version                   version.OperatorVersion
	Maturity                  string
	CustomResourceDefinitions CustomResourceDefinitions
	APIServiceDefinitions     APIServiceDefinitions
	NativeAPIs                []metav1.GroupVersionKind
	MinKubeVersion            string
	DisplayName               string
	Description               string
	Keywords                  []string
	Maintainers               []Maintainer
	Provider                  AppLink
	Links                     []AppLink
	Icon                      []Icon

	// InstallModes specify supported installation types
	// +optional
	InstallModes []InstallMode

	// The name of a CSV this one replaces. Should match the `metadata.Name` field of the old CSV.
	// +optional
	Replaces string

	// Map of string keys and values that can be used to organize and categorize
	// (scope and select) objects.
	// +optional
	Labels map[string]string

	// Annotations is an unstructured key value map stored with a resource that may be
	// set by external tools to store and retrieve arbitrary metadata.
	// +optional
	Annotations map[string]string

	// Label selector for related resources.
	// +optional
	Selector *metav1.LabelSelector
}

type Maintainer struct {
	Name  string
	Email string
}

type AppLink struct {
	Name string
	URL  string
}

type Icon struct {
	Data      string
	MediaType string
}

// ClusterServiceVersionPhase is a label for the condition of a ClusterServiceVersion at the current time.
type ClusterServiceVersionPhase string

// These are the valid phases of ClusterServiceVersion
const (
	CSVPhaseNone = ""
	// CSVPhasePending means the csv has been accepted by the system, but the install strategy has not been attempted.
	// This is likely because there are unmet requirements.
	CSVPhasePending ClusterServiceVersionPhase = "Pending"
	// CSVPhaseInstallReady means that the requirements are met but the install strategy has not been run.
	CSVPhaseInstallReady ClusterServiceVersionPhase = "InstallReady"
	// CSVPhaseInstalling means that the install strategy has been initiated but not completed.
	CSVPhaseInstalling ClusterServiceVersionPhase = "Installing"
	// CSVPhaseSucceeded means that the resources in the CSV were created successfully.
	CSVPhaseSucceeded ClusterServiceVersionPhase = "Succeeded"
	// CSVPhaseFailed means that the install strategy could not be successfully completed.
	CSVPhaseFailed ClusterServiceVersionPhase = "Failed"
	// CSVPhaseUnknown means that for some reason the state of the csv could not be obtained.
	CSVPhaseUnknown ClusterServiceVersionPhase = "Unknown"
	// CSVPhaseReplacing means that a newer CSV has been created and the csv's resources will be transitioned to a new owner.
	CSVPhaseReplacing ClusterServiceVersionPhase = "Replacing"
	// CSVPhaseDeleting means that a CSV has been replaced by a new one and will be checked for safety before being deleted
	CSVPhaseDeleting ClusterServiceVersionPhase = "Deleting"
	// CSVPhaseAny matches all other phases in CSV queries
	CSVPhaseAny ClusterServiceVersionPhase = ""
)

// ConditionReason is a camelcased reason for the state transition
type ConditionReason string

const (
	CSVReasonRequirementsUnknown                         ConditionReason = "RequirementsUnknown"
	CSVReasonRequirementsNotMet                          ConditionReason = "RequirementsNotMet"
	CSVReasonRequirementsMet                             ConditionReason = "AllRequirementsMet"
	CSVReasonOwnerConflict                               ConditionReason = "OwnerConflict"
	CSVReasonComponentFailed                             ConditionReason = "InstallComponentFailed"
	CSVReasonInvalidStrategy                             ConditionReason = "InvalidInstallStrategy"
	CSVReasonWaiting                                     ConditionReason = "InstallWaiting"
	CSVReasonInstallSuccessful                           ConditionReason = "InstallSucceeded"
	CSVReasonInstallCheckFailed                          ConditionReason = "InstallCheckFailed"
	CSVReasonComponentUnhealthy                          ConditionReason = "ComponentUnhealthy"
	CSVReasonBeingReplaced                               ConditionReason = "BeingReplaced"
	CSVReasonReplaced                                    ConditionReason = "Replaced"
	CSVReasonNeedsReinstall                              ConditionReason = "NeedsReinstall"
	CSVReasonNeedsCertRotation                           ConditionReason = "NeedsCertRotation"
	CSVReasonAPIServiceResourceIssue                     ConditionReason = "APIServiceResourceIssue"
	CSVReasonAPIServiceResourcesNeedReinstall            ConditionReason = "APIServiceResourcesNeedReinstall"
	CSVReasonAPIServiceInstallFailed                     ConditionReason = "APIServiceInstallFailed"
	CSVReasonCopied                                      ConditionReason = "Copied"
	CSVReasonInvalidInstallModes                         ConditionReason = "InvalidInstallModes"
	CSVReasonNoTargetNamespaces                          ConditionReason = "NoTargetNamespaces"
	CSVReasonUnsupportedOperatorGroup                    ConditionReason = "UnsupportedOperatorGroup"
	CSVReasonNoOperatorGroup                             ConditionReason = "NoOperatorGroup"
	CSVReasonTooManyOperatorGroups                       ConditionReason = "TooManyOperatorGroups"
	CSVReasonInterOperatorGroupOwnerConflict             ConditionReason = "InterOperatorGroupOwnerConflict"
	CSVReasonCannotModifyStaticOperatorGroupProvidedAPIs ConditionReason = "CannotModifyStaticOperatorGroupProvidedAPIs"
)

// Conditions appear in the status as a record of state transitions on the ClusterServiceVersion
type ClusterServiceVersionCondition struct {
	// Condition of the ClusterServiceVersion
	Phase ClusterServiceVersionPhase
	// A human readable message indicating details about why the ClusterServiceVersion is in this condition.
	// +optional
	Message string
	// A brief CamelCase message indicating details about why the ClusterServiceVersion is in this state.
	// e.g. 'RequirementsNotMet'
	// +optional
	Reason ConditionReason
	// Last time we updated the status
	// +optional
	LastUpdateTime metav1.Time
	// Last time the status transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time
}

// OwnsCRD determines whether the current CSV owns a paritcular CRD.
func (csv ClusterServiceVersion) OwnsCRD(name string) bool {
	for _, desc := range csv.Spec.CustomResourceDefinitions.Owned {
		if desc.Name == name {
			return true
		}
	}

	return false
}

// OwnsAPIService determines whether the current CSV owns a paritcular APIService.
func (csv ClusterServiceVersion) OwnsAPIService(name string) bool {
	for _, desc := range csv.Spec.APIServiceDefinitions.Owned {
		apiServiceName := fmt.Sprintf("%s.%s", desc.Version, desc.Group)
		if apiServiceName == name {
			return true
		}
	}

	return false
}

// StatusReason is a camelcased reason for the status of a RequirementStatus or DependentStatus
type StatusReason string

const (
	RequirementStatusReasonPresent             StatusReason = "Present"
	RequirementStatusReasonNotPresent          StatusReason = "NotPresent"
	RequirementStatusReasonPresentNotSatisfied StatusReason = "PresentNotSatisfied"
	// The CRD is present but the Established condition is False (not available)
	RequirementStatusReasonNotAvailable StatusReason = "PresentNotAvailable"
	DependentStatusReasonSatisfied      StatusReason = "Satisfied"
	DependentStatusReasonNotSatisfied   StatusReason = "NotSatisfied"
)

// DependentStatus is the status for a dependent requirement (to prevent infinite nesting)
type DependentStatus struct {
	Group   string
	Version string
	Kind    string
	Status  StatusReason
	UUID    string
	Message string
}

type RequirementStatus struct {
	Group      string
	Version    string
	Kind       string
	Name       string
	Status     StatusReason
	Message    string
	UUID       string
	Dependents []DependentStatus
}

// ClusterServiceVersionStatus represents information about the status of a pod. Status may trail the actual
// state of a system.
type ClusterServiceVersionStatus struct {
	// Current condition of the ClusterServiceVersion
	Phase ClusterServiceVersionPhase
	// A human readable message indicating details about why the ClusterServiceVersion is in this condition.
	// +optional
	Message string
	// A brief CamelCase message indicating details about why the ClusterServiceVersion is in this state.
	// e.g. 'RequirementsNotMet'
	// +optional
	Reason ConditionReason
	// Last time we updated the status
	// +optional
	LastUpdateTime metav1.Time
	// Last time the status transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time
	// List of conditions, a history of state transitions
	Conditions []ClusterServiceVersionCondition
	// The status of each requirement for this CSV
	RequirementStatus []RequirementStatus
	// Last time the owned APIService certs were updated
	// +optional
	CertsLastUpdated metav1.Time
	// Time the owned APIService certs will rotate next
	// +optional
	CertsRotateAt metav1.Time
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient

// ClusterServiceVersion is a Custom Resource of type `ClusterServiceVersionSpec`.
type ClusterServiceVersion struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec   ClusterServiceVersionSpec
	Status ClusterServiceVersionStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterServiceVersionList represents a list of ClusterServiceVersions.
type ClusterServiceVersionList struct {
	metav1.TypeMeta
	metav1.ListMeta

	Items []ClusterServiceVersion
}

// GetAllCRDDescriptions returns a deduplicated set of CRDDescriptions that is
// the union of the owned and required CRDDescriptions.
//
// Descriptions with the same name prefer the value in Owned.
// Descriptions are returned in alphabetical order.
func (csv ClusterServiceVersion) GetAllCRDDescriptions() []CRDDescription {
	set := make(map[string]CRDDescription)
	for _, required := range csv.Spec.CustomResourceDefinitions.Required {
		set[required.Name] = required
	}

	for _, owned := range csv.Spec.CustomResourceDefinitions.Owned {
		set[owned.Name] = owned
	}

	keys := make([]string, 0)
	for key := range set {
		keys = append(keys, key)
	}
	sort.StringSlice(keys).Sort()

	descs := make([]CRDDescription, 0)
	for _, key := range keys {
		descs = append(descs, set[key])
	}

	return descs
}

// GetAllAPIServiceDescriptions returns a deduplicated set of APIServiceDescriptions that is
// the union of the owned and required APIServiceDescriptions.
//
// Descriptions with the same name prefer the value in Owned.
// Descriptions are returned in alphabetical order.
func (csv ClusterServiceVersion) GetAllAPIServiceDescriptions() []APIServiceDescription {
	set := make(map[string]APIServiceDescription)
	for _, required := range csv.Spec.APIServiceDefinitions.Required {
		name := fmt.Sprintf("%s.%s", required.Version, required.Group)
		set[name] = required
	}

	for _, owned := range csv.Spec.APIServiceDefinitions.Owned {
		name := fmt.Sprintf("%s.%s", owned.Version, owned.Group)
		set[name] = owned
	}

	keys := make([]string, 0)
	for key := range set {
		keys = append(keys, key)
	}
	sort.StringSlice(keys).Sort()

	descs := make([]APIServiceDescription, 0)
	for _, key := range keys {
		descs = append(descs, set[key])
	}

	return descs
}

// GetRequiredAPIServiceDescriptions returns a deduplicated set of required APIServiceDescriptions
// with the intersection of required and owned removed
// Equivalent to the set subtraction required - owned
//
// Descriptions are returned in alphabetical order.
func (csv ClusterServiceVersion) GetRequiredAPIServiceDescriptions() []APIServiceDescription {
	set := make(map[string]APIServiceDescription)
	for _, required := range csv.Spec.APIServiceDefinitions.Required {
		name := fmt.Sprintf("%s.%s", required.Version, required.Group)
		set[name] = required
	}

	// Remove any shared owned from the set
	for _, owned := range csv.Spec.APIServiceDefinitions.Owned {
		name := fmt.Sprintf("%s.%s", owned.Version, owned.Group)
		if _, ok := set[name]; ok {
			delete(set, name)
		}
	}

	keys := make([]string, 0)
	for key := range set {
		keys = append(keys, key)
	}
	sort.StringSlice(keys).Sort()

	descs := make([]APIServiceDescription, 0)
	for _, key := range keys {
		descs = append(descs, set[key])
	}

	return descs
}

// GetOwnedAPIServiceDescriptions returns a deduplicated set of owned APIServiceDescriptions
//
// Descriptions are returned in alphabetical order.
func (csv ClusterServiceVersion) GetOwnedAPIServiceDescriptions() []APIServiceDescription {
	set := make(map[string]APIServiceDescription)
	for _, owned := range csv.Spec.APIServiceDefinitions.Owned {
		name := owned.GetName()
		set[name] = owned
	}

	keys := make([]string, 0)
	for key := range set {
		keys = append(keys, key)
	}
	sort.StringSlice(keys).Sort()

	descs := make([]APIServiceDescription, 0)
	for _, key := range keys {
		descs = append(descs, set[key])
	}

	return descs
}
