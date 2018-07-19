package e2eutil

import (
	"strings"
	"testing"

	y2j "github.com/ghodss/yaml"
	yaml "gopkg.in/yaml.v2"
	apps "k8s.io/api/apps/v1"
	"k8s.io/api/rbac/v1beta1"
	crd "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	extensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	extensions_scheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

func GetCRClient(t *testing.T, config *rest.Config, yamlCR []byte) *rest.RESTClient {
	// get new RESTClient for custom resources
	crConfig := config
	m := make(map[interface{}]interface{})
	err := yaml.Unmarshal(yamlCR, &m)
	groupVersion := strings.Split(m["apiVersion"].(string), "/")
	crGV := schema.GroupVersion{Group: groupVersion[0], Version: groupVersion[1]}
	crConfig.GroupVersion = &crGV
	crConfig.APIPath = "/apis"
	crConfig.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}

	if crConfig.UserAgent == "" {
		crConfig.UserAgent = rest.DefaultKubernetesUserAgent()
	}
	crRESTClient, err := rest.RESTClientFor(crConfig)
	if err != nil {
		t.Fatal(err)
	}
	return crRESTClient
}

func createCRFromYAML(t *testing.T, yamlFile []byte, kubeconfig *rest.Config, namespace, resourceName string) error {
	client := GetCRClient(t, kubeconfig, yamlFile)
	jsonDat, err := y2j.YAMLToJSON(yamlFile)
	err = client.Post().
		Namespace(namespace).
		Resource(resourceName).
		Body(jsonDat).
		Do().
		Error()
	return err
}

func createCRDFromYAML(t *testing.T, yamlFile []byte, extensionsClient *extensions.Clientset) error {
	decode := extensions_scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode(yamlFile, nil, nil)

	if err != nil {
		t.Log("Failed to deserialize CustomResourceDefinition")
		t.Fatal(err)
	}
	switch o := obj.(type) {
	case *crd.CustomResourceDefinition:
		_, err = extensionsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(o)
		return err
	}
	return nil
}

func CreateFromYAML(t *testing.T, yamlFile []byte, kubeclient *kubernetes.Clientset, kubeconfig *rest.Config, namespace string) error {
	m := make(map[interface{}]interface{})
	err := yaml.Unmarshal(yamlFile, &m)
	kind := m["kind"].(string)
	switch kind {
	case "Role":
	case "RoleBinding":
	case "Deployment":
	case "CustomResourceDefinition":
		extensionclient, err := extensions.NewForConfig(kubeconfig)
		if err != nil {
			t.Fatal(err)
		}
		return createCRDFromYAML(t, yamlFile, extensionclient)
	// we assume that all custom resources are from operator-sdk and thus follow
	// a common naming convention
	default:
		return createCRFromYAML(t, yamlFile, kubeconfig, namespace, strings.ToLower(kind)+"s")
	}
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode(yamlFile, nil, nil)

	if err != nil {
		t.Log("Unable to deserialize resource; is it a custom resource?")
		t.Fatal(err)
	}

	switch o := obj.(type) {
	case *v1beta1.Role:
		_, err = kubeclient.RbacV1beta1().Roles(namespace).Create(o)
		return err
	case *v1beta1.RoleBinding:
		_, err = kubeclient.RbacV1beta1().RoleBindings(namespace).Create(o)
		return err
	case *apps.Deployment:
		_, err = kubeclient.AppsV1().Deployments(namespace).Create(o)
		return err
	default:
		t.Fatalf("unknown type: %s", o)
	}
	return nil
}
