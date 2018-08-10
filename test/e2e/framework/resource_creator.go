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
	"errors"
	"strings"

	y2j "github.com/ghodss/yaml"
	yaml "gopkg.in/yaml.v2"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/api/rbac/v1beta1"
	crd "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	extensions_scheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

func (ctx *TestCtx) createCRFromYAML(yamlFile []byte, resourceName string) error {
	client, err := ctx.GetCRClient(yamlFile)
	if err != nil {
		return err
	}
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return err
	}
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

func (ctx *TestCtx) CreateFromYAML(yamlFile []byte) error {
	yamlMap := make(map[interface{}]interface{})
	err := yaml.Unmarshal(yamlFile, &yamlMap)
	if err != nil {
		return err
	}
	kind := yamlMap["kind"].(string)
	switch kind {
	case "Role":
	case "RoleBinding":
	case "Deployment":
	case "CustomResourceDefinition":
		return ctx.createCRDFromYAML(yamlFile)
	// we assume that all custom resources are from operator-sdk and thus follow
	// a common naming convention
	default:
		return ctx.createCRFromYAML(yamlFile, strings.ToLower(kind)+"s")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode(yamlFile, nil, nil)
	if err != nil {
		return err
	}

	namespace, err := ctx.GetNamespace()
	if err != nil {
		return err
	}
	switch o := obj.(type) {
	case *v1beta1.Role:
		_, err = Global.KubeClient.RbacV1beta1().Roles(namespace).Create(o)
		ctx.AddFinalizerFn(func() error {
			return Global.KubeClient.RbacV1beta1().Roles(namespace).Delete(o.Name, metav1.NewDeleteOptions(0))
		})
		return err
	case *v1beta1.RoleBinding:
		_, err = Global.KubeClient.RbacV1beta1().RoleBindings(namespace).Create(o)
		ctx.AddFinalizerFn(func() error {
			return Global.KubeClient.RbacV1beta1().RoleBindings(namespace).Delete(o.Name, metav1.NewDeleteOptions(0))
		})
		return err
	case *apps.Deployment:
		_, err = Global.KubeClient.AppsV1().Deployments(namespace).Create(o)
		ctx.AddFinalizerFn(func() error {
			return Global.KubeClient.AppsV1().Deployments(namespace).Delete(o.Name, metav1.NewDeleteOptions(0))
		})
		return err
	default:
		return errors.New("unhandled resource type")
	}
}
