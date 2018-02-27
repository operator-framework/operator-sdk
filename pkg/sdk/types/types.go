package types

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
)

// Object is the Kubernetes runtime.Object interface expected
// of all resources that the user can watch.
type Object runtime.Object

// Event is triggered when some change has happened on the watched resources.
// If created or updated, Object would be the current state and Deleted=false.
// If deleted, Object would be the last known state and Deleted=true.
type Event struct {
	Object  Object
	Deleted bool
}

// Context is the special context that is passed to the Handler.
// It includes:
// - Context: standard Go context that is used to pass cancellation signals and deadlines
type Context struct {
	Context context.Context
}

// FuncType defines the type of the function of an Action.
type FuncType string

// KubeFunc is the function signature for supported kubernetes functions
type KubeFunc func(Object) error

// Action defines what Function to apply on a given Object.
type Action struct {
	Object Object
	Func   FuncType
}
