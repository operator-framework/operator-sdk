package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

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
var kubeconfig *rest.Config

func main() {
	os.Chdir(os.Getenv("GOPATH") + "/src/github.com/example-inc")
	fmt.Println("Creating new operator project")
	cmdOut, err := exec.Command("operator-sdk",
		"new",
		"memcached-operator",
		"--api-version=cache.example.com/v1alpha1",
		"--kind=Memcached").CombinedOutput()
	if err != nil {
		log.Fatalf("Error: %v\nCommand Output: %s\n", err, string(cmdOut))
	}
	os.Chdir("memcached-operator")
	os.RemoveAll("vendor/github.com/operator-framework/operator-sdk/pkg")
	os.Symlink(os.Getenv("TRAVIS_BUILD_DIR")+"/pkg", "vendor/github.com/operator-framework/operator-sdk/pkg")
	handler, err := os.Create("pkg/stub/handler.go")
	if err != nil {
		log.Fatal(err)
	}
	defer handler.Close()
	resp, err := http.Get("https://raw.githubusercontent.com/operator-framework/operator-sdk/master/example/memcached-operator/handler.go.tmpl")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	_, err = io.Copy(handler, resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	gotypes, err := ioutil.ReadFile("pkg/apis/cache/v1alpha1/types.go")
	if err != nil {
		log.Fatal(err)
	}
	lines := bytes.Split(gotypes, []byte("\n"))
	lines = lines[:len(lines)-7]
	lines = append(lines, []byte("type MemcachedSpec struct {	Size int32 `json:\"size\"`}"))
	lines = append(lines, []byte("type MemcachedStatus struct {Nodes []string `json:\"nodes\"`}\n"))
	os.Remove("pkg/apis/cache/v1alpha1/types.go")
	err = ioutil.WriteFile("pkg/apis/cache/v1alpha1/types.go", bytes.Join(lines, []byte("\n")), os.FileMode(int(0664)))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Generating k8s")
	cmdOut, err = exec.Command("operator-sdk",
		"generate",
		"k8s").CombinedOutput()
	if err != nil {
		log.Fatalf("Error: %v\nCommand Output: %s\n", err, string(cmdOut))
	}

	fmt.Println("Building operator docker image")
	cmdOut, err = exec.Command("operator-sdk",
		"build",
		"quay.io/example/memcached-operator:v0.0.1").CombinedOutput()
	if err != nil {
		log.Fatalf("Error: %v\nCommand Output: %s\n", err, string(cmdOut))
	}

	opYAML, err := ioutil.ReadFile("deploy/operator.yaml")
	if err != nil {
		log.Fatal(err)
	}
	opYAML = bytes.Replace(opYAML, []byte("imagePullPolicy: Always"), []byte("imagePullPolicy: Never"), 1)
	err = ioutil.WriteFile("deploy/operator.yaml", opYAML, os.FileMode(int(0664)))
	if err != nil {
		log.Fatal(err)
	}

	namespace := "memcached"
	kubeconfigPath := filepath.Join(
		os.Getenv("HOME"), ".kube", "config",
	)
	kubeconfig, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		log.Fatal(err)
	}
	kubeclient, err := kubernetes.NewForConfig(kubeconfig)
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
		err = createFromYAML(thing, kubeclient, namespace)
		if err != nil {
			log.Fatal(err)
		}
	}
	//	kubectlWrapper("create", namespace, "deploy/rbac.yaml")
	fmt.Println("Created rbac")

	// create crd
	yamlCRD, err := ioutil.ReadFile("deploy/crd.yaml")
	//	kubectlWrapper("create", namespace, "deploy/crd.yaml")
	err = createFromYAML(yamlCRD, kubeclient, namespace)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Created crd")

	// create operator
	dat, err = ioutil.ReadFile("deploy/operator.yaml")
	//kubectlWrapper("create", namespace, "deploy/operator.yaml")
	err = createFromYAML(dat, kubeclient, namespace)
	if err != nil {
		log.Fatal(err)
	}
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

	yamlCR, err := ioutil.ReadFile("deploy/cr.yaml")
	memcachedClient := getCRClient(kubeconfig, yamlCR)
	createFromYAML(yamlCR, kubeclient, namespace)

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

	err = memcachedClient.Delete().
		Namespace(namespace).
		Resource("memcacheds").
		Name("example-memcached").
		Body([]byte("{\"propagationPolicy\":\"Foreground\"}")).
		Do().
		Error()
	if err != nil {
		fmt.Println("Failed to delete example-memcached CR")
		log.Fatal(err)
	}
	err = kubeclient.AppsV1().Deployments(namespace).
		Delete("memcached-operator", nil)
	if err != nil {
		fmt.Println("Failed to delete memcached-operator deployment")
		log.Fatal(err)
	}
}

func getCRClient(config *rest.Config, yamlCR []byte) *rest.RESTClient {
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
		log.Fatal(err)
	}
	return crRESTClient
}

// create a custom resource from a yaml file; not fully automated (still needs more work)
func createCRFromYAML(yamlFile []byte, namespace, resourceName string) error {
	client := getCRClient(kubeconfig, yamlFile)
	jsonDat, err := y2j.YAMLToJSON(yamlFile)
	err = client.Post().
		Namespace(namespace).
		Resource(resourceName).
		Body(jsonDat).
		Do().
		Error()
	return err
}

func createCRDFromYAML(yamlFile []byte, extensionsClient *extensions.Clientset) error {
	decode := extensions_scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode(yamlFile, nil, nil)

	if err != nil {
		fmt.Println("Failed to deserialize CustomResourceDefinition")
		log.Fatal(err)
	}
	switch o := obj.(type) {
	case *crd.CustomResourceDefinition:
		_, err = extensionsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(o)
		return err
	}
	return nil
}

func createFromYAML(yamlFile []byte, kubeclient *kubernetes.Clientset, namespace string) error {
	m := make(map[interface{}]interface{})
	err := yaml.Unmarshal(yamlFile, &m)
	kind := m["kind"].(string)
	switch kind {
	case "Role":
		fallthrough
	case "RoleBinding":
		fallthrough
	case "Deployment":
	case "CustomResourceDefinition":
		extensionclient, err := extensions.NewForConfig(kubeconfig)
		if err != nil {
			log.Fatal(err)
		}
		return createCRDFromYAML(yamlFile, extensionclient)
	case "Memcached":
		return createCRFromYAML(yamlFile, namespace, "memcacheds")
	}
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode(yamlFile, nil, nil)

	if err != nil {
		fmt.Println("Unable to deserialize resource; is it a custom resource?")
		log.Fatal(err)
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
		log.Fatalf("unknown type: %s", o)
	}
	return nil
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
	output, err := exec.Command("kubectl", action, "--namespace="+namespace, "-f", file).CombinedOutput()
	if err != nil {
		fmt.Println("An error occurred")
		fmt.Printf("%s\n", output)
		log.Fatalf("%s\n", err)
	}
}
