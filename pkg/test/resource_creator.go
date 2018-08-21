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
	"bytes"
	goctx "context"
	"fmt"
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (ctx *TestCtx) GetNamespace() (string, error) {
	if ctx.Namespace != "" {
		return ctx.Namespace, nil
	}
	// create namespace
	ctx.Namespace = ctx.GetID()
	namespaceObj := &core.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ctx.Namespace}}
	_, err := Global.KubeClient.CoreV1().Namespaces().Create(namespaceObj)
	if apierrors.IsAlreadyExists(err) {
		return "", fmt.Errorf("namespace %s already exists: %v", ctx.Namespace, err)
	} else if err != nil {
		return "", err
	}
	ctx.AddFinalizerFn(func() error {
		return Global.KubeClient.CoreV1().Namespaces().Delete(ctx.Namespace, metav1.NewDeleteOptions(0))
	})
	return ctx.Namespace, nil
}

func setNamespaceYAML(yamlFile []byte, namespace string) ([]byte, error) {
	yamlMap := make(map[interface{}]interface{})
	err := yaml.Unmarshal(yamlFile, &yamlMap)
	if err != nil {
		return nil, err
	}
	yamlMap["metadata"].(map[interface{}]interface{})["namespace"] = namespace
	return yaml.Marshal(yamlMap)
}

func (ctx *TestCtx) createFromYAML(yamlFile []byte, skipIfExists bool) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return err
	}
	yamlSplit := bytes.Split(yamlFile, []byte("\n---\n"))
	for _, yamlSpec := range yamlSplit {
		yamlSpec, err = setNamespaceYAML(yamlSpec, namespace)
		if err != nil {
			return err
		}

		obj, _, err := Global.DynamicDecoder.Decode(yamlSpec, nil, nil)
		if err != nil {
			return err
		}

		err = Global.DynamicClient.Create(goctx.TODO(), obj)
		if skipIfExists && apierrors.IsAlreadyExists(err) {
			continue
		}
		if err != nil {
			return err
		}
		ctx.AddFinalizerFn(func() error { return Global.DynamicClient.Delete(goctx.TODO(), obj) })
	}
	return nil
}

func (ctx *TestCtx) InitializeClusterResources() error {
	// create namespaced resources
	namespacedYAML, err := ioutil.ReadFile(*Global.NamespacedManPath)
	if err != nil {
		return fmt.Errorf("failed to read namespaced manifest: %v", err)
	}
	return ctx.createFromYAML(namespacedYAML, false)
}
