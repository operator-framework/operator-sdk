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

package sdk

import (
	"fmt"

	"github.com/operator-framework/operator-sdk/pkg/k8sclient"
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Create creates the provided object on the server and updates the arg
// "object" with the result from the server(UID, resourceVersion, etc).
// Returns an error if the object’s TypeMeta(Kind, APIVersion) or ObjectMeta(Name/GenerateName, Namespace) is missing or incorrect.
// Can also return an api error from the server
// e.g AlreadyExists https://github.com/kubernetes/apimachinery/blob/master/pkg/api/errors/errors.go#L423
func Create(object Object) (err error) {
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

// Patch patches provided "object" on the server with given "patch" and updates the arg
// "object" with the result from the server(UID, resourceVersion, etc).
// Returns an error if the object’s TypeMeta(Kind, APIVersion) or ObjectMeta(Name, Namespace) is missing or incorrect.
// Returns an error if patch couldn't be json serialized into bytes.
// Can also return an api error from the server
// e.g Conflict https://github.com/kubernetes/apimachinery/blob/master/pkg/api/errors/errors.go#L428
func Patch(object Object, pt types.PatchType, patch []byte) (err error) {
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

	unstructObj, err := resourceClient.Patch(name, pt, patch)
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
func Update(object Object) (err error) {
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

// DeleteOp wraps all the options for Delete().
type DeleteOp struct {
	metaDeleteOptions *metav1.DeleteOptions
}

// DeleteOption configures DeleteOp.
type DeleteOption func(*DeleteOp)

func NewDeleteOp() *DeleteOp {
	op := &DeleteOp{}
	op.setDefaults()
	return op
}

func (op *DeleteOp) applyOpts(opts []DeleteOption) {
	for _, opt := range opts {
		opt(op)
	}
}

func (op *DeleteOp) setDefaults() {
	if op.metaDeleteOptions == nil {
		op.metaDeleteOptions = &metav1.DeleteOptions{}
	}
}

// WithDeleteOptions sets the metav1.DeleteOptions for the Delete() operation.
func WithDeleteOptions(metaDeleteOptions *metav1.DeleteOptions) DeleteOption {
	return func(op *DeleteOp) {
		op.metaDeleteOptions = metaDeleteOptions
	}
}

// Delete deletes the specified object
// Returns an error if the object’s TypeMeta(Kind, APIVersion) or ObjectMeta(Name, Namespace) is missing or incorrect.
// e.g NotFound https://github.com/kubernetes/apimachinery/blob/master/pkg/api/errors/errors.go#L418
// “opts” configures the DeleteOptions
// When passed WithDeleteOptions(o), the specified metav1.DeleteOptions are set.
func Delete(object Object, opts ...DeleteOption) (err error) {
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
