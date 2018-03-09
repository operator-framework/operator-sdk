// Copyright 2018 The Operator-SDK Authors
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

const (
	// Supported function types
	KubeApplyFunc  sdkTypes.FuncType = "kube-apply"
	KubeDeleteFunc sdkTypes.FuncType = "kube-delete"
)

var (
	// kubeFuncs is the mapping of the supported functions
	kubeFuncs = map[sdkTypes.FuncType]sdkTypes.KubeFunc{
		KubeApplyFunc:  KubeApply,
		KubeDeleteFunc: KubeDelete,
	}
)

// ProcessAction invokes the function specified by action.Func
func ProcessAction(action sdkTypes.Action) error {
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

// KubeApply tries to create the specified object or update it if it already exists
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

	unstructObj := k8sutil.UnstructuredFromRuntimeObject(object)

	// Create the resource if it doesn't exist
	_, err = resourceClient.Create(unstructObj)
	if err == nil {
		return nil
	}
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create object (%s): %v ", objectInfo, err)
	}

	// Update it if it already exists
	// NOTE: The update could fail if there is a resourceVersion conflict.
	// That means the object is stale, and the user needs to retry the Action with
	// an updated object that has the latest resourceVersion
	_, err = resourceClient.Update(unstructObj)
	if err != nil {
		return fmt.Errorf("failed to update object (%s): %v ", objectInfo, err)
	}
	return nil
}

// KubeDelete deletes an object if it still exists
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
	if err != nil && !apierrors.IsNotFound(err) {
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
