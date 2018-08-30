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

package framework

import (
	"bytes"
	goctx "context"
	"strings"

	y2j "github.com/ghodss/yaml"
	yaml "gopkg.in/yaml.v2"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

func (ctx *TestCtx) GetNamespace() (string, error) {
	if ctx.Namespace != "" {
		return ctx.Namespace, nil
	}
	// create namespace
	ctx.Namespace = ctx.GetID()
	namespaceObj := &core.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ctx.Namespace}}
	_, err := Global.KubeClient.CoreV1().Namespaces().Create(namespaceObj)
	if err != nil {
		return "", err
	}
	ctx.AddFinalizerFn(func() error {
		return Global.KubeClient.CoreV1().Namespaces().Delete(ctx.Namespace, metav1.NewDeleteOptions(0))
	})
	return ctx.Namespace, nil
}

func (ctx *TestCtx) GetCRClient(yamlCR []byte) (*rest.RESTClient, error) {
	if ctx.CRClient != nil {
		return ctx.CRClient, nil
	}
	// get new RESTClient for custom resources
	crConfig := Global.KubeConfig
	yamlMap := make(map[interface{}]interface{})
	err := yaml.Unmarshal(yamlCR, &yamlMap)
	if err != nil {
		return nil, err
	}
	groupVersion := strings.Split(yamlMap["apiVersion"].(string), "/")
	crGV := schema.GroupVersion{Group: groupVersion[0], Version: groupVersion[1]}
	crConfig.GroupVersion = &crGV
	crConfig.APIPath = "/apis"
	crConfig.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}

	if crConfig.UserAgent == "" {
		crConfig.UserAgent = rest.DefaultKubernetesUserAgent()
	}
	ctx.CRClient, err = rest.RESTClientFor(crConfig)
	if err != nil {
		return nil, err
	}
	return ctx.CRClient, nil
}

func (ctx *TestCtx) createCRFromYAML(yamlFile []byte) error {
	client, err := ctx.GetCRClient(yamlFile)
	if err != nil {
		return err
	}
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return err
	}
	yamlMap := make(map[interface{}]interface{})
	err = yaml.Unmarshal(yamlFile, &yamlMap)
	if err != nil {
		return err
	}
	kind := yamlMap["kind"].(string)
	resourceName := kind + "s"
	jsonDat, err := y2j.YAMLToJSON(yamlFile)
	err = client.Post().
		Namespace(namespace).
		Resource(resourceName).
		Body(jsonDat).
		Do().
		Error()
	ctx.AddFinalizerFn(func() error {
		return client.Delete().
			Namespace(namespace).
			Resource(resourceName).
			Body(metav1.NewDeleteOptions(0)).
			Do().
			Error()
	})
	return err
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

func (ctx *TestCtx) CreateFromYAML(yamlFile []byte) error {
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
			// DynamicClient/DynamicDecoder can only handle standard and extensions kubernetes resources.
			// If a resource is not recognized by the decoder, assume it's a custom resource and fall back
			// to createCRFromYAML.
			return ctx.createCRFromYAML(yamlFile)
		}

		err = Global.DynamicClient.Create(goctx.TODO(), obj)
		if err != nil {
			return err
		}
		ctx.AddFinalizerFn(func() error { return Global.DynamicClient.Delete(goctx.TODO(), obj) })
	}
	return nil
}
