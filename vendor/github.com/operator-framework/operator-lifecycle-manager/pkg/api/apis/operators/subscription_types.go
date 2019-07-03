package operators

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// SubscriptionKind is the PascalCase name of a Subscription's kind.
const SubscriptionKind = "SubscriptionKind"

// SubscriptionState tracks when updates are available, installing, or service is up to date
type SubscriptionState string

const (
	SubscriptionStateNone             = ""
	SubscriptionStateFailed           = "UpgradeFailed"
	SubscriptionStateUpgradeAvailable = "UpgradeAvailable"
	SubscriptionStateUpgradePending   = "UpgradePending"
	SubscriptionStateAtLatest         = "AtLatestKnown"
)

const (
	SubscriptionReasonInvalidCatalog   ConditionReason = "InvalidCatalog"
	SubscriptionReasonUpgradeSucceeded ConditionReason = "UpgradeSucceeded"
)

// SubscriptionSpec defines an Application that can be installed
type SubscriptionSpec struct {
	CatalogSource          string
	CatalogSourceNamespace string
	Package                string
	Channel                string
	StartingCSV            string
	InstallPlanApproval    Approval
}

type SubscriptionStatus struct {
	// CurrentCSV is the CSV the Subscription is progressing to.
	// +optional
	CurrentCSV string

	// InstalledCSV is the CSV currently installed by the Subscription.
	// +optional
	InstalledCSV string

	// Install is a reference to the latest InstallPlan generated for the Subscription.
	// DEPRECATED: InstallPlanRef
	// +optional
	Install *InstallPlanReference

	// State represents the current state of the Subscription
	// +optional
	State SubscriptionState

	// Reason is the reason the Subscription was transitioned to its current state.
	// +optional
	Reason ConditionReason

	// InstallPlanRef is a reference to the latest InstallPlan that contains the Subscription's current CSV.
	// +optional
	InstallPlanRef *corev1.ObjectReference

	// LastUpdated represents the last time that the Subscription status was updated.
	LastUpdated metav1.Time
}

type InstallPlanReference struct {
	APIVersion string
	Kind       string
	Name       string
	UID        types.UID
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient

// Subscription keeps operators up to date by tracking changes to Catalogs.
type Subscription struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec   *SubscriptionSpec
	Status SubscriptionStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SubscriptionList is a list of Subscription resources.
type SubscriptionList struct {
	metav1.TypeMeta
	metav1.ListMeta

	Items []Subscription
}

// GetInstallPlanApproval gets the configured install plan approval or the default
func (s *Subscription) GetInstallPlanApproval() Approval {
	if s.Spec.InstallPlanApproval == ApprovalManual {
		return ApprovalManual
	}
	return ApprovalAutomatic
}

// NewInstallPlanReference returns an InstallPlanReference for the given ObjectReference.
func NewInstallPlanReference(ref *corev1.ObjectReference) *InstallPlanReference {
	return &InstallPlanReference{
		APIVersion: ref.APIVersion,
		Kind:       ref.Kind,
		Name:       ref.Name,
		UID:        ref.UID,
	}
}
