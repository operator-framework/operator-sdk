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

package client

import (
	"errors"
	"io"
	"strings"

	"github.com/operator-framework/operator-lib/handler"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"helm.sh/helm/v3/pkg/kube"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ kube.Interface = &ownerRefInjectingClient{}

func NewOwnerRefInjectingClient(base kube.Interface, restMapper meta.RESTMapper,
	obj client.Object) (kube.Interface, error) {

	if obj != nil {
		if obj.GetObjectKind() != nil {
			if obj.GetObjectKind().GroupVersionKind().Empty() || obj.GetName() == "" || obj.GetUID() == "" {
				var err = errors.New("owner resource is invalid")
				return nil, err
			}
		}
	}
	return &ownerRefInjectingClient{
		Interface:  base,
		restMapper: restMapper,
		owner:      obj,
	}, nil
}

type ownerRefInjectingClient struct {
	kube.Interface
	restMapper meta.RESTMapper
	owner      client.Object
}

func (c *ownerRefInjectingClient) Build(reader io.Reader, validate bool) (kube.ResourceList, error) {
	resourceList, err := c.Interface.Build(reader, validate)
	if err != nil {
		return resourceList, err
	}
	err = resourceList.Visit(func(r *resource.Info, err error) error {
		if err != nil {
			return err
		}
		objMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(r.Object)
		if err != nil {
			return err
		}
		u := &unstructured.Unstructured{Object: objMap}
		useOwnerRef, err := k8sutil.SupportsOwnerReference(c.restMapper, c.owner, u, "")
		if err != nil {
			return err
		}

		// If the resource contains the Helm resource-policy keep annotation, then do not add
		// the owner reference. So when the CR is deleted, Kubernetes won't GCs the resource.
		if useOwnerRef && !containsResourcePolicyKeep(u.GetAnnotations()) {
			ownerRef := metav1.NewControllerRef(c.owner, c.owner.GetObjectKind().GroupVersionKind())
			u.SetOwnerReferences([]metav1.OwnerReference{*ownerRef})
		} else {
			err := handler.SetOwnerAnnotations(u, c.owner)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resourceList, nil
}

func containsResourcePolicyKeep(annotations map[string]string) bool {
	if annotations == nil {
		return false
	}
	resourcePolicyType, ok := annotations[kube.ResourcePolicyAnno]
	if !ok {
		return false
	}
	resourcePolicyType = strings.ToLower(strings.TrimSpace(resourcePolicyType))
	return resourcePolicyType == kube.KeepPolicy
}
