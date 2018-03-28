# Operator-SDK Action API Design Doc

## Goal

Define an API for performing Actions(Create,Update,Delete) that is flexible enough to allow the SDK user to execute a sequence of dependent Actions and perform error handling.

## Background

For milestone `0.0.1` the SDK invokes the user provided handler for an Event, which would return a list of Actions that the SDK would execute. The motivation for this was to ensure that all Actions are taken through the SDK.

```Go
// Handle reacts to events and outputs actions.
// If any intended action failed, the event would be re-triggered.
// For actions done before the failed action, there is no rollback.
func Handle(ctx context.Context, event sdkTypes.Event) []sdkTypes.Action
```

The provided actions were:
- `kube-apply`: Create or Update or specified object
- `kube-delete`: Delete the specified object

This workflow of batch executing actions outside of the handler has the following issues:
It makes it harder for the user to write their operator logic as a sequence of dependent actions and queries, e.g:
- Create a ConfigMap
- If it already exists
  - Query some application state and verify the ConfigMap data
  - Update the ConfigMap data with the latest state if needed

The user cannot handle errors for failed actions e.g updating the status if some action fails. Currently the Handler would just be retriggered with the Event if an Action fails

## Proposed Solution

The SDK should provide an API to Create, Update and Delete objects from inside the Handler. This will allow users to write the core operator logic in a more intuitive way by using a sequence of actions and queries.

This method also aligns with the original goal of ensuring that all actions of the operator are taken through the SDK.

## API
**Note:** `sdkTypes.Object` is just the kubernetes `runtime.Object` 

### Handler:

```Go
// Handle contains the business logic for an handling an Event
// It uses SDK Actions and Queries to reconcile the state
// If an error is returned the Event would be requeued and sent to the Handler again
func Handle(ctx context.Context, event sdkTypes.Event) error
```

### Create Update Delete:
```Go
// Create creates the provided object on the server, and updates 
// the local object with the generated result from the server(UID, resourceVersion, etc).
// Returns an error if the object’s TypeMeta(Kind, APIVersion) or ObjectMeta(Name, Namespace) is missing or incorrect.
// Can also return an api error from the server
// e.g AlreadyExists https://github.com/kubernetes/apimachinery/blob/master/pkg/api/errors/errors.go#L423 
func Create(object sdkTypes.Object) error
```

```Go
// Update updates the provided object on the server, and updates
// the local object with the generated result from the server(UID, resourceVersion, etc).
// Returns an error if the object’s TypeMeta(Kind, APIVersion) or ObjectMeta(Name, Namespace) is missing or incorrect.
// Can also return an api error from the server
// e.g Conflict https://github.com/kubernetes/apimachinery/blob/master/pkg/api/errors/errors.go#L428 
func Update(object sdkTypes.Object) error
```

```Go
// Delete deletes the specified object.
// Returns an error if the object’s TypeMeta(Kind, APIVersion) or ObjectMeta(Name, Namespace) is missing or incorrect.
// Can also return an api error from the server
// e.g NotFound https://github.com/kubernetes/apimachinery/blob/master/pkg/api/errors/errors.go#L418
// “opts” configures the k8s.io/apimachinery/pkg/apis/meta/v1.DeleteOptions
func Delete(object sdkTypes.Object, opts ...DeleteOption) error
```

### Example Usage:

```Go
import (
    "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func Handle(ctx context.Context, event sdkType.Event) {
    ...
    pod := &v1.Pod{
        TypeMeta: metav1.TypeMeta{
            Kind:       "Pod",
            APIVersion: "v1",
        },
        ObjectMeta: metav1.ObjectMeta {
            Name:      "example",
            Namespace: "default",
        },
        Spec: v1.PodSpec{
            ...
        }
    }

    // Create
    err := sdk.Create(pod)
    if err != nil && !apierrors.IsAlreadyExists(err) {
        return errors.New("failed to create pod")
    }

    // Update
    err := sdk.Update(pod)
    if err != nil {
        return errors.New("failed to update pod")
    }

    ...
	
    // Delete with default options
    err := sdk.Delete(pod)
    if err != nil {
        return errors.New("failed to delete pod")
    }
    
    // Delete with custom options
    gracePeriodSeconds := int64(5)
    metav1DeleteOptions := &metav1.DeleteOptions{GracePeriodSeconds: &gracePeriodSeconds}
    err := sdk.Delete(pod, sdk.WithDeleteOptions(metav1DeleteOptions))
}
```

## Reference:

### DeleteOptions:

```Go
// DeleteOp wraps all the options for Delete.
type DeleteOp struct {
    metaDeleteOptions metav1.DeleteOptions
}

// DeleteOption configures DeleteOp
type DeleteOption func(*DeleteOp) 

// WithDeleteOptions sets the meta_v1.DeleteOptions for
// the Delete() operation.
func WithDeleteOptions(metaDeleteOptions *metav1.DeleteOptions) DeleteOption {
    return func(op *DeleteOp) {
        op.metaDeleteOptions = metaDeleteOptions
    }
}
```


