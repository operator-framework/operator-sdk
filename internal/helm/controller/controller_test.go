// Copyright 2026 The Operator-SDK Authors
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

package controller

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	rpb "helm.sh/helm/v3/pkg/release"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"

	libhandler "github.com/operator-framework/operator-lib/handler"
)

type fakeManager struct {
	manager.Manager
	restMapper meta.RESTMapper
}

func (m *fakeManager) GetRESTMapper() meta.RESTMapper { return m.restMapper }
func (m *fakeManager) GetScheme() *runtime.Scheme     { return runtime.NewScheme() }
func (m *fakeManager) GetCache() cache.Cache          { return nil }

type fakeController struct {
	controller.Controller
	sources []source.Source
}

func (c *fakeController) Watch(src source.Source) error {
	c.sources = append(c.sources, src)
	return nil
}

func TestWatchDependentResources(t *testing.T) {
	ownerGVK := schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Nginx"}
	restMapper := meta.NewDefaultRESTMapper(nil)
	restMapper.Add(ownerGVK, meta.RESTScopeNamespace)
	restMapper.Add(schema.GroupVersionKind{Version: "v1", Kind: "ConfigMap"}, meta.RESTScopeNamespace)

	cases := []struct {
		name              string
		manifest          string
		expectAnnotations bool
	}{
		{
			name: "Uses owner references when the dependent omits its namespace",
			manifest: `apiVersion: v1
kind: ConfigMap
metadata:
  name: cm`,
			expectAnnotations: false,
		},
		{
			name: "Uses owner references when the dependent is in the release namespace",
			manifest: `apiVersion: v1
kind: ConfigMap
metadata:
  name: cm
  namespace: ns`,
			expectAnnotations: false,
		},
		{
			name: "Uses annotations when the dependent is in another namespace",
			manifest: `apiVersion: v1
kind: ConfigMap
metadata:
  name: cm
  namespace: other`,
			expectAnnotations: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := &HelmOperatorReconciler{GVK: ownerGVK}
			ctr := &fakeController{}
			watchDependentResources(&fakeManager{restMapper: restMapper}, r, ctr)

			err := r.releaseHook(&rpb.Release{Namespace: "ns", Manifest: c.manifest})
			assert.NoError(t, err)
			if assert.Len(t, ctr.sources, 1) {
				assert.Equal(t, c.expectAnnotations, isAnnotationHandler(ctr.sources[0]))
			}
		})
	}
}

func sourceHandler(src source.Source) handler.EventHandler {
	return reflect.ValueOf(src).Elem().FieldByName("Handler").Interface().(handler.EventHandler)
}

func isAnnotationHandler(src source.Source) bool {
	h := sourceHandler(src)
	_, ok := h.(*libhandler.EnqueueRequestForAnnotation[client.Object])
	return ok
}
