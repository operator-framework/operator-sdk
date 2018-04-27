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

	"github.com/operator-framework/operator-sdk/pkg/k8sclient"
	sdkTypes "github.com/operator-framework/operator-sdk/pkg/sdk/types"
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
)

// Create creates the provided object on the server and updates the arg
// "object" with the result from the server(UID, resourceVersion, etc).
// Returns an error if the object’s TypeMeta(Kind, APIVersion) or ObjectMeta(Name/GenerateName, Namespace) is missing or incorrect.
// Can also return an api error from the server
// e.g AlreadyExists https://github.com/kubernetes/apimachinery/blob/master/pkg/api/errors/errors.go#L423
func Create(object sdkTypes.Object) (err error) {
	_, namespace, err := k8sutil.GetNameAndNamespace(object)
	if err != nil {
		return err
	}
	gvk := object.GetObjectKind().GroupVersionKind()
	apiVersion, kind := gvk.ToAPIVersionAndKind()

	resourceClient, _, err := k8sclient.GetResourceClient(apiVersion, kind, namespace)
	if err != nil {
		return fmt.Errorf("failed to get resource client: %v", err)
	}

	unstructObj := k8sutil.UnstructuredFromRuntimeObject(object)
	unstructObj, err = resourceClient.Create(unstructObj)
	if err != nil {
		return err
	}

	// Update the arg object with the result
	err = k8sutil.UnstructuredIntoRuntimeObject(unstructObj, object)
	if err != nil {
		return fmt.Errorf("failed to unmarshal the retrieved data: %v", err)
	}
	return nil
}

// Update updates the provided object on the server and updates the arg
// "object" with the result from the server(UID, resourceVersion, etc).
// Returns an error if the object’s TypeMeta(Kind, APIVersion) or ObjectMeta(Name, Namespace) is missing or incorrect.
// Can also return an api error from the server
// e.g Conflict https://github.com/kubernetes/apimachinery/blob/master/pkg/api/errors/errors.go#L428
func Update(object sdkTypes.Object) (err error) {
	_, namespace, err := k8sutil.GetNameAndNamespace(object)
	if err != nil {
		return err
	}
	gvk := object.GetObjectKind().GroupVersionKind()
	apiVersion, kind := gvk.ToAPIVersionAndKind()

	resourceClient, _, err := k8sclient.GetResourceClient(apiVersion, kind, namespace)
	if err != nil {
		return fmt.Errorf("failed to get resource client: %v", err)
	}

	unstructObj := k8sutil.UnstructuredFromRuntimeObject(object)
	unstructObj, err = resourceClient.Update(unstructObj)
	if err != nil {
		return err
	}

	// Update the arg object with the result
	err = k8sutil.UnstructuredIntoRuntimeObject(unstructObj, object)
	if err != nil {
		return fmt.Errorf("failed to unmarshal the retrieved data: %v", err)
	}
	return nil
}

// Delete deletes the specified object
// Returns an error if the object’s TypeMeta(Kind, APIVersion) or ObjectMeta(Name, Namespace) is missing or incorrect.
// e.g NotFound https://github.com/kubernetes/apimachinery/blob/master/pkg/api/errors/errors.go#L418
// “opts” configures the DeleteOptions
// When passed WithDeleteOptions(o), the specified metav1.DeleteOptions are set.
func Delete(object sdkTypes.Object, opts ...DeleteOption) (err error) {
	name, namespace, err := k8sutil.GetNameAndNamespace(object)
	if err != nil {
		return err
	}
	gvk := object.GetObjectKind().GroupVersionKind()
	apiVersion, kind := gvk.ToAPIVersionAndKind()

	resourceClient, _, err := k8sclient.GetResourceClient(apiVersion, kind, namespace)
	if err != nil {
		return fmt.Errorf("failed to get resource client: %v", err)
	}

	o := NewDeleteOp()
	o.applyOpts(opts)
	return resourceClient.Delete(name, o.metaDeleteOptions)
}
