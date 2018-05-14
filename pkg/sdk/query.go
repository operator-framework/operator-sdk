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
)

// Get gets the specified object and unmarshals the retrieved data into the "into" object.
// "into" is a Object that must have
// "Kind" and "APIVersion" specified in its "TypeMeta" field
// and "Name" and "Namespace" specified in its "ObjectMeta" field.
// "opts" configures the Get operation.
//  When passed With WithGetOptions(o), the specified metav1.GetOptions is set.
func Get(into Object, opts ...GetOption) error {
	name, namespace, err := k8sutil.GetNameAndNamespace(into)
	if err != nil {
		return err
	}
	gvk := into.GetObjectKind().GroupVersionKind()
	apiVersion, kind := gvk.ToAPIVersionAndKind()
	resourceClient, _, err := k8sclient.GetResourceClient(apiVersion, kind, namespace)
	if err != nil {
		return fmt.Errorf("failed to get resource client for (apiVersion:%s, kind:%s, ns:%s): %v", apiVersion, kind, namespace, err)
	}
	o := NewGetOp()
	o.applyOpts(opts)
	u, err := resourceClient.Get(name, *o.metaGetOptions)
	if err != nil {
		return err
	}
	if err := k8sutil.UnstructuredIntoRuntimeObject(u, into); err != nil {
		return fmt.Errorf("failed to unmarshal the retrieved data: %v", err)
	}
	return nil
}

// List retrieves the specified object list and unmarshals the retrieved data into the "into" object.
// "namespace" indicates which kubernetes namespace to look for the list of kubernetes objects.
// "into" is a sdkType.Object that must have
// "Kind" and "APIVersion" specified in its "TypeMeta" field
// "opts" configures the List operation.
//  When passed With WithListOptions(o), the specified metav1.ListOptions is set.
func List(namespace string, into Object, opts ...ListOption) error {
	gvk := into.GetObjectKind().GroupVersionKind()
	apiVersion, kind := gvk.ToAPIVersionAndKind()
	resourceClient, _, err := k8sclient.GetResourceClient(apiVersion, kind, namespace)
	if err != nil {
		return fmt.Errorf("failed to get resource client for (apiVersion:%s, kind:%s, ns:%s): %v", apiVersion, kind, namespace, err)
	}
	o := NewListOp()
	o.applyOpts(opts)
	l, err := resourceClient.List(*o.metaListOptions)
	if err != nil {
		return err
	}
	if err := k8sutil.RuntimeObjectIntoRuntimeObject(l, into); err != nil {
		return fmt.Errorf("failed to unmarshal the retrieved data: %v", err)
	}
	return nil
}
