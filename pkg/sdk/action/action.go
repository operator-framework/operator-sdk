package action

import (
	"fmt"

	"github.com/coreos/operator-sdk/pkg/k8sclient"
	sdkTypes "github.com/coreos/operator-sdk/pkg/sdk/types"
	"github.com/coreos/operator-sdk/pkg/util/k8sutil"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FuncName defines the name of the function of an Action.
type FuncName string

// FuncType is the function signature for supported kubernetes functions
type FuncType func(sdkTypes.Object) error

// Action defines what Function to apply on a given Object.
type Action struct {
	Object sdkTypes.Object
	Func   FuncName
}

const (
	// Function names for supported functions
	KubeApplyFunc  FuncName = "kube-apply"
	KubeDeleteFunc FuncName = "kube-delete"
)

var (
	// kubeFuncs is the mapping of the supported functions
	kubeFuncs = map[FuncName]FuncType{
		KubeApplyFunc:  KubeApply,
		KubeDeleteFunc: KubeDelete,
	}
)

// ProcessAction invokes the function specified by action.Func
func ProcessAction(action Action) error {
	kubeFunc, ok := kubeFuncs[action.Func]
	if !ok {
		return fmt.Errorf("failed to process action: unsupported function (%v)", action.Func)
	}
	err := kubeFunc(action.Object)
	if err != nil {
		return fmt.Errorf("failed to process action: %v", err)
	}
	return nil
}

// KubeApply will try to create the specified object or update it if it already exists
func KubeApply(object sdkTypes.Object) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("kube-apply failed: %v", err)
		}
	}()

	name, namespace, err := getNameAndNamespace(object)
	if err != nil {
		return err
	}
	gvk := object.GetObjectKind().GroupVersionKind()
	apiVersion, kind := gvk.ToAPIVersionAndKind()
	objectInfo := objectInfoString(kind, name, namespace)

	resourceClient, _, err := k8sclient.GetResourceClient(apiVersion, kind, namespace)
	if err != nil {
		return fmt.Errorf("failed to get resource client for object: %v", err)
	}

	unstructObj, err := k8sutil.UnstructuredFromRuntimeObject(object)
	if err != nil {
		return fmt.Errorf("failed to get runtime object from unstructured: %v", err)
	}

	// Create the resource if it doesn't exist
	_, err = resourceClient.Create(unstructObj)
	if err == nil {
		return nil
	}
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create object (%s): %v ", objectInfo, err)
	}

	// Update it if it already exists
	// TODO: Should resourceVersion conflict be handled differently
	_, err = resourceClient.Update(unstructObj)
	if err != nil {
		return fmt.Errorf("failed to update object (%s): %v ", objectInfo, err)
	}
	return nil
}

// KubeDelete deletes an object
func KubeDelete(object sdkTypes.Object) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("kube-delete failed: %v", err)
		}
	}()

	name, namespace, err := getNameAndNamespace(object)
	if err != nil {
		return err
	}
	gvk := object.GetObjectKind().GroupVersionKind()
	apiVersion, kind := gvk.ToAPIVersionAndKind()
	objectInfo := objectInfoString(kind, name, namespace)

	resourceClient, _, err := k8sclient.GetResourceClient(apiVersion, kind, namespace)
	if err != nil {
		return fmt.Errorf("failed to get resource client for object: %v", err)
	}

	err = resourceClient.Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object (%s): %v", objectInfo, err)
	}
	return nil
}

func getNameAndNamespace(object sdkTypes.Object) (string, string, error) {
	accessor := meta.NewAccessor()
	name, err := accessor.Name(object)
	if err != nil {
		return "", "", fmt.Errorf("failed to get name for object: %v", err)
	}
	namespace, err := accessor.Namespace(object)
	if err != nil {
		return "", "", fmt.Errorf("failed to get namespace for object: %v", err)
	}
	return name, namespace, nil
}

func objectInfoString(kind, name, namespace string) string {
	return kind + ": " + namespace + "/" + name
}
