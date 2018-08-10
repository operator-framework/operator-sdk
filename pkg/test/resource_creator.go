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
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	y2j "github.com/ghodss/yaml"
	yaml "gopkg.in/yaml.v2"
	core "k8s.io/api/core/v1"
	crd "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	extensions_scheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
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

func (ctx *TestCtx) GetCRClient(yamlCR []byte) (*rest.RESTClient, error) {
	if ctx.CRClient != nil {
		return ctx.CRClient, nil
	}
	// a user may pass nil if they expect the CRClient to already exist
	if yamlCR == nil {
		return nil, errors.New("ctx.CRClient does not exist; yamlCR cannot be nil")
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
	return ctx.CRClient, err
}

// TODO: Implement a way for a user to add their own scheme to us the dynamic
// client to eliminate the need for the UpdateCR function

// UpdateCR takes the name of a resource, the resource plural name,
// the path of the field that need to be updated (e.g. /spec/size),
// and the new value to that field and patches the resource with
// that change
func (ctx *TestCtx) UpdateCR(name, resourceName, path, value string) error {
	crClient, err := ctx.GetCRClient(nil)
	if err != nil {
		return err
	}
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return err
	}
	return crClient.Patch(types.JSONPatchType).
		Namespace(namespace).
		Resource(resourceName).
		Name(name).
		Body([]byte("[{\"op\": \"replace\", \"path\": \"" + path + "\", \"value\": " + value + "}]")).
		Do().
		Error()
}

func (ctx *TestCtx) createCRFromYAML(yamlFile []byte, resourceName string) error {
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
	// TODO: handle failure of this line without segfault
	name := yamlMap["metadata"].(map[interface{}]interface{})["name"].(string)
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
			Name(name).
			Body(metav1.NewDeleteOptions(0)).
			Do().
			Error()
	})
	return err
}

func (ctx *TestCtx) createCRDFromYAML(yamlFile []byte) error {
	decode := extensions_scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode(yamlFile, nil, nil)
	if err != nil {
		return err
	}
	switch o := obj.(type) {
	case *crd.CustomResourceDefinition:
		_, err = Global.ExtensionsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(o)
		ctx.AddFinalizerFn(func() error {
			err = Global.ExtensionsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(o.Name, metav1.NewDeleteOptions(0))
			if err != nil && !apierrors.IsNotFound(err) {
				return err
			}
			return nil
		})
		if apierrors.IsAlreadyExists(err) {
			return nil
		}
		return err
	default:
		return errors.New("non-CRD resource in createCRDFromYAML function")
	}
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
			yamlMap := make(map[interface{}]interface{})
			err = yaml.Unmarshal(yamlSpec, &yamlMap)
			if err != nil {
				return err
			}
			kind := yamlMap["kind"].(string)
			err = ctx.createCRFromYAML(yamlSpec, strings.ToLower(kind)+"s")
			if err != nil {
				return err
			}
			continue
		}

		err = Global.DynamicClient.Create(goctx.TODO(), obj)
		if err != nil {
			return err
		}
		ctx.AddFinalizerFn(func() error { return Global.DynamicClient.Delete(goctx.TODO(), obj) })
	}
	return nil
}

func (ctx *TestCtx) InitializeClusterResources() error {
	// create rbac
	rbacYAML, err := ioutil.ReadFile(*Global.RbacManPath)
	if err != nil {
		return fmt.Errorf("failed to read rbac manifest: %v", err)
	}
	err = ctx.CreateFromYAML(rbacYAML)
	if err != nil {
		return err
	}
	// create operator deployment
	operatorYAML, err := ioutil.ReadFile(*Global.OpManPath)
	if err != nil {
		return fmt.Errorf("failed to read operator manifest: %v", err)
	}
	return ctx.CreateFromYAML(operatorYAML)
}
