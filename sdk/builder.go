package sdk

type Builder interface {
	// Name is the name of the operator.
	Name(name string) Builder
	// CRDName is the full name of the Custom Resource Resource that the operator operates on.
	CRDName(crdName string) Builder
	// APIVersion is the version from the groupVersion.
	APIVersion(apiVersion string) Builder
	// Callbacks registers the callbacks which notifies the creation, update, and deletion of any Custom Resource.
	Callbacks(callBacks CallBacks) Builder
	// Build builds an operator instance.
	Build() (Operator, error)
}

type CallBacks interface {
	// OnAdd notifies a creation of a Custom Resource.
	OnAdd(obj interface{})
	// OnUpdate notifies a update of an existing Custom Resource.
	OnUpdate(oldObj, newObj interface{})
	// OnDelete notifies a deletion of an existing Custom Resource.
	OnDelete(obj interface{})
}

// NewOperatorBuilder creates a operator builder.
func NewOperatorBuilder() Builder {
	// Implement me!
	return nil
}
