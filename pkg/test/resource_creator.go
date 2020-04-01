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

package test

import (
	goctx "context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"

	"github.com/ghodss/yaml"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
)

// TODO: remove before 1.0.0
// Deprecated: GetNamespace() exists for historical compatibility.
// Use GetOperatorNamespace() or GetWatchNamespace() instead
func (ctx *Context) GetNamespace() (string, error) {
	if ctx.namespace != "" {
		return ctx.namespace, nil
	}
	// create namespace
	ctx.namespace = ctx.GetID()
	namespaceObj := &core.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ctx.namespace}}
	_, err := ctx.kubeclient.CoreV1().Namespaces().Create(namespaceObj)
	if apierrors.IsAlreadyExists(err) {
		return "", fmt.Errorf("namespace %s already exists: %w", ctx.namespace, err)
	} else if err != nil {
		return "", err
	}
	ctx.AddCleanupFn(func() error {
		return ctx.kubeclient.CoreV1().Namespaces().Delete(ctx.namespace, metav1.NewDeleteOptions(0))
	})
	return ctx.namespace, nil
}

// GetOperatorNamespace will return an Operator Namespace,
// if the flag --operator-namespace  not be used (TestOpeatorNamespaceEnv not set)
// then it will create a new namespace with randon name and return that namespace
func (ctx *Context) GetOperatorNamespace() (string, error) {
	if ctx.operatorNamespace != "" {
		return ctx.operatorNamespace, nil
	}
	// create namespace
	ctx.operatorNamespace = ctx.GetID()
	namespaceObj := &core.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ctx.operatorNamespace}}
	_, err := ctx.kubeclient.CoreV1().Namespaces().Create(namespaceObj)
	if apierrors.IsAlreadyExists(err) {
		return "", fmt.Errorf("namespace %s already exists: %w", ctx.operatorNamespace, err)
	} else if err != nil {
		return "", err
	}
	ctx.AddCleanupFn(func() error {
		return ctx.kubeclient.CoreV1().Namespaces().Delete(ctx.operatorNamespace, metav1.NewDeleteOptions(0))
	})
	return ctx.operatorNamespace, nil
}

// GetWatchNamespace will return the  namespaces to operator
// watch for changes, if the flag --watch-namespaced not be used
// then it will  return the Operator Namespace.
func (ctx *Context) GetWatchNamespace() (string, error) {
	// if ctx.watchNamespace is already set and not "";
	// then return ctx.watchnamespace
	if ctx.watchNamespace != "" {
		return ctx.watchNamespace, nil
	}
	// if ctx.watchNamespace == "";
	// ensure it was set explicitly using TestWatchNamespaceEnv
	if ns, ok := os.LookupEnv(TestWatchNamespaceEnv); ok {
		return ns, nil
	}
	// get ctx.operatorNamespace (use ctx.GetOperatorNamespace()
	// to make sure ctx.operatorNamespace is not "")
	operatorNamespace, err := ctx.GetOperatorNamespace()
	if err != nil {
		return "", nil
	}
	ctx.watchNamespace = operatorNamespace
	return ctx.watchNamespace, nil
}

func (ctx *Context) createFromYAML(yamlFile []byte, skipIfExists bool, cleanupOptions *CleanupOptions) error {
	operatorNamespace, err := ctx.GetOperatorNamespace()
	if err != nil {
		return err
	}
	scanner := k8sutil.NewYAMLScanner(yamlFile)
	for scanner.Scan() {
		yamlSpec := scanner.Bytes()

		obj := &unstructured.Unstructured{}
		jsonSpec, err := yaml.YAMLToJSON(yamlSpec)
		if err != nil {
			return fmt.Errorf("could not convert yaml file to json: %w", err)
		}
		if err := obj.UnmarshalJSON(jsonSpec); err != nil {
			return fmt.Errorf("failed to unmarshal object spec: %w", err)
		}
		obj.SetNamespace(operatorNamespace)
		err = ctx.client.Create(goctx.TODO(), obj, cleanupOptions)
		if skipIfExists && apierrors.IsAlreadyExists(err) {
			continue
		}
		if err != nil {
			_, restErr := ctx.restMapper.RESTMappings(obj.GetObjectKind().GroupVersionKind().GroupKind())
			if restErr == nil {
				return err
			}
			// don't store error, as only error will be timeout. Error from runtime client will be easier for
			// the user to understand than the timeout error, so just use that if we fail
			_ = wait.PollImmediate(time.Second*1, time.Second*10, func() (bool, error) {
				ctx.restMapper.Reset()
				_, err := ctx.restMapper.RESTMappings(obj.GetObjectKind().GroupVersionKind().GroupKind())
				if err != nil {
					return false, nil
				}
				return true, nil
			})
			err = ctx.client.Create(goctx.TODO(), obj, cleanupOptions)
			if skipIfExists && apierrors.IsAlreadyExists(err) {
				continue
			}
			if err != nil {
				return err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to scan manifest: %w", err)
	}
	return nil
}

func (ctx *Context) InitializeClusterResources(cleanupOptions *CleanupOptions) error {
	// create namespaced resources
	namespacedYAML, err := ioutil.ReadFile(ctx.namespacedManPath)
	if err != nil {
		return fmt.Errorf("failed to read namespaced manifest: %w", err)
	}
	return ctx.createFromYAML(namespacedYAML, false, cleanupOptions)
}
