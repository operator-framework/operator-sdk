package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	y2j "github.com/ghodss/yaml"
	"github.com/operator-framework/operator-sdk/pkg/util/retryutil"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/api/rbac/v1beta1"
	crd "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	extensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	extensions_scheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var retryInterval = time.Second * 5

func main() {
	namespace := "memcached"
	kubeconfig := filepath.Join(
		os.Getenv("HOME"), ".kube", "config",
	)
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	kubeclient, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	extensionclient, err := extensions.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	// create namespace
	namespaceObj := &core.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	_, err = kubeclient.CoreV1().Namespaces().Create(namespaceObj)
	if err != nil {
		log.Fatal(err)
	}

	// create rbac
	dat, err := ioutil.ReadFile("deploy/rbac.yaml")
	splitDat := bytes.Split(dat, []byte("\n---\n"))
	for _, thing := range splitDat {
		createFromYAML(thing, kubeclient, namespace)
	}
	//	kubectlWrapper("create", namespace, "deploy/rbac.yaml")
	fmt.Println("Created rbac")

	// create crd
	dat, err = ioutil.ReadFile("deploy/crd.yaml")
	//	kubectlWrapper("create", namespace, "deploy/crd.yaml")
	createCRDFromYAML(dat, extensionclient)
	fmt.Println("Created crd")

	// create operator
	dat, err = ioutil.ReadFile("deploy/operator.yaml")
	//kubectlWrapper("create", namespace, "deploy/operator.yaml")
	createFromYAML(dat, kubeclient, namespace)
	fmt.Println("Created operator")

	err = deploymentReplicaCheck(kubeclient, namespace, "memcached-operator", 1, 6)
	if err != nil {
		log.Fatal(err)
	}

	// create example-memcached yaml file
	file, err := os.OpenFile("deploy/cr.yaml", os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	_, err = file.WriteString("apiVersion: \"cache.example.com/v1alpha1\"\nkind: \"Memcached\"\nmetadata:\n  name: \"example-memcached\"\nspec:\n  size: 3")
	if err != nil {
		log.Fatal(err)
	}

	file.Close()

	dat, err = ioutil.ReadFile("deploy/cr.yaml")
	memcachedClient := getCRClient(config, "cache.example.com", "v1alpha1")
	createCRFromYAML(dat, memcachedClient, namespace, "memcacheds")

	//kubectlWrapper("apply", namespace, "deploy/cr.yaml")

	err = deploymentReplicaCheck(kubeclient, namespace, "example-memcached", 3, 6)
	if err != nil {
		log.Fatal(err)
	}

	// update CR size to 4
	err = memcachedClient.Patch(types.JSONPatchType).
		Namespace(namespace).
		Resource("memcacheds").
		Name("example-memcached").
		Body([]byte("[{\"op\": \"replace\", \"path\": \"/spec/size\", \"value\": 4}]")).
		Do().
		Error()
	if err != nil {
		log.Fatal(err)
	}

	err = deploymentReplicaCheck(kubeclient, namespace, "example-memcached", 4, 6)
	if err != nil {
		log.Fatal(err)
	}

	kubectlWrapper("delete", namespace, "deploy/cr.yaml")
	kubectlWrapper("delete", namespace, "deploy/operator.yaml")
}

func getCRClient(config *rest.Config, group, version string) *rest.RESTClient {
	// get new RESTClient for custom resources
	crConfig := config
	crGV := schema.GroupVersion{Group: group, Version: version}
	crConfig.GroupVersion = &crGV
	crConfig.APIPath = "/apis"
	crConfig.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}

	if crConfig.UserAgent == "" {
		crConfig.UserAgent = rest.DefaultKubernetesUserAgent()
	}
	crRESTClient, err := rest.RESTClientFor(crConfig)
	if err != nil {
		log.Fatal(err)
	}
	return crRESTClient
}

// create a custom resource from a yaml file; not fully automated (still needs more work)
func createCRFromYAML(yaml []byte, client *rest.RESTClient, namespace, resourceName string) {
	jsonDat, err := y2j.YAMLToJSON(yaml)
	err = client.Post().
		Namespace(namespace).
		Resource(resourceName).
		Body(jsonDat).
		Do().
		Error()
	if err != nil {
		log.Fatal(err)
	}
}

func createCRDFromYAML(yaml []byte, extensionsClient *extensions.Clientset) {
	decode := extensions_scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode(yaml, nil, nil)

	if err != nil {
		fmt.Println("Failed to deserialize CustomResourceDefinition")
		log.Fatal(err)
	}
	switch o := obj.(type) {
	case *crd.CustomResourceDefinition:
		extensionsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(o)
	}
}

func createFromYAML(yaml []byte, kubeclient *kubernetes.Clientset, namespace string) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode(yaml, nil, nil)

	if err != nil {
		fmt.Println("Unable to deserialize resource; is it a custom resource?")
		log.Fatal(err)
	}

	switch o := obj.(type) {
	case *v1beta1.Role:
		kubeclient.RbacV1beta1().Roles(namespace).Create(o)
	case *v1beta1.RoleBinding:
		kubeclient.RbacV1beta1().RoleBindings(namespace).Create(o)
	case *apps.Deployment:
		kubeclient.AppsV1().Deployments(namespace).Create(o)
	default:
		log.Fatalf("unknown type: %s", o)
	}
}

func printDeployments(deployments *apps.DeploymentList) {
	template := "%-40s%-10s\n"
	fmt.Printf(template, "NAME", "NUM_REPLICAS")
	for _, deployment := range deployments.Items {
		fmt.Printf(
			template,
			deployment.Name,
			strconv.Itoa(int(deployment.Status.AvailableReplicas)),
		)
	}
}

func deploymentReplicaCheck(kubeclient *kubernetes.Clientset, namespace, name string, replicas, retries int) error {
	err := retryutil.Retry(retryInterval, retries, func() (done bool, err error) {
		deployment, err := kubeclient.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{IncludeUninitialized: true})
		if err != nil {
			// sometimes, a deployment has not been created by the time we call this; we
			// assume that is what happened instead of immediately failing
			return false, nil
		}

		if int(deployment.Status.AvailableReplicas) == replicas {
			return true, nil
		}
		fmt.Printf("Waiting for full availability of %s deployment (%d/%d)\n", name, deployment.Status.AvailableReplicas, replicas)
		return false, nil
	})
	if err != nil {
		return err
	}
	fmt.Printf("Deployment available (%d/%d)\n", replicas, replicas)
	return nil
}

func kubectlWrapper(action, namespace, file string) {
	output, err := exec.Command("kubectl", action, "--namespace="+namespace, "-f", file).Output()
	if err != nil {
		fmt.Println("An error occurred")
		fmt.Printf("%s\n", output)
		log.Fatalf("%s\n", err)
	}
}
