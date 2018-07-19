package e2e

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/operator-framework/operator-sdk/test/e2e/e2eutil"
	core "k8s.io/api/core/v1"
	extensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func TestMemcached(t *testing.T) {
	os.Chdir(os.Getenv("GOPATH") + "/src/github.com/example-inc")
	t.Log("Creating new operator project")
	cmdOut, err := exec.Command("operator-sdk",
		"new",
		"memcached-operator",
		"--api-version=cache.example.com/v1alpha1",
		"--kind=Memcached").CombinedOutput()
	if err != nil {
		t.Fatalf("Error: %v\nCommand Output: %s\n", err, string(cmdOut))
	}
	os.Chdir("memcached-operator")
	os.RemoveAll("vendor/github.com/operator-framework/operator-sdk/pkg")
	os.Symlink(os.Getenv("TRAVIS_BUILD_DIR")+"/pkg", "vendor/github.com/operator-framework/operator-sdk/pkg")
	handler, err := os.Create("pkg/stub/handler.go")
	if err != nil {
		t.Fatal(err)
	}
	defer handler.Close()
	resp, err := http.Get("https://raw.githubusercontent.com/operator-framework/operator-sdk/master/example/memcached-operator/handler.go.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, err = io.Copy(handler, resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	gotypes, err := ioutil.ReadFile("pkg/apis/cache/v1alpha1/types.go")
	if err != nil {
		t.Fatal(err)
	}
	lines := bytes.Split(gotypes, []byte("\n"))
	lines = lines[:len(lines)-7]
	lines = append(lines, []byte("type MemcachedSpec struct {	Size int32 `json:\"size\"`}"))
	lines = append(lines, []byte("type MemcachedStatus struct {Nodes []string `json:\"nodes\"`}\n"))
	os.Remove("pkg/apis/cache/v1alpha1/types.go")
	err = ioutil.WriteFile("pkg/apis/cache/v1alpha1/types.go", bytes.Join(lines, []byte("\n")), os.FileMode(int(0664)))
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Generating k8s")
	cmdOut, err = exec.Command("operator-sdk",
		"generate",
		"k8s").CombinedOutput()
	if err != nil {
		t.Fatalf("Error: %v\nCommand Output: %s\n", err, string(cmdOut))
	}

	t.Log("Building operator docker image")
	cmdOut, err = exec.Command("operator-sdk",
		"build",
		"quay.io/example/memcached-operator:v0.0.1").CombinedOutput()
	if err != nil {
		t.Fatalf("Error: %v\nCommand Output: %s\n", err, string(cmdOut))
	}

	opYAML, err := ioutil.ReadFile("deploy/operator.yaml")
	if err != nil {
		t.Fatal(err)
	}
	opYAML = bytes.Replace(opYAML, []byte("imagePullPolicy: Always"), []byte("imagePullPolicy: Never"), 1)
	err = ioutil.WriteFile("deploy/operator.yaml", opYAML, os.FileMode(int(0664)))
	if err != nil {
		t.Fatal(err)
	}

	namespace := "memcached"
	kubeconfigPath := filepath.Join(
		os.Getenv("HOME"), ".kube", "config",
	)
	kubeconfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		t.Fatal(err)
	}
	kubeclient, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		t.Fatal(err)
	}

	// create namespace
	namespaceObj := &core.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	_, err = kubeclient.CoreV1().Namespaces().Create(namespaceObj)
	if err != nil {
		t.Fatal(err)
	}

	// create rbac
	dat, err := ioutil.ReadFile("deploy/rbac.yaml")
	splitDat := bytes.Split(dat, []byte("\n---\n"))
	for _, thing := range splitDat {
		err = e2eutil.CreateFromYAML(t, thing, kubeclient, kubeconfig, namespace)
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Log("Created rbac")

	// create crd
	yamlCRD, err := ioutil.ReadFile("deploy/crd.yaml")
	err = e2eutil.CreateFromYAML(t, yamlCRD, kubeclient, kubeconfig, namespace)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Created crd")

	// create operator
	dat, err = ioutil.ReadFile("deploy/operator.yaml")
	err = e2eutil.CreateFromYAML(t, dat, kubeclient, kubeconfig, namespace)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Created operator")

	err = e2eutil.DeploymentReplicaCheck(t, kubeclient, namespace, "memcached-operator", 1, 6)
	if err != nil {
		t.Fatal(err)
	}

	// create example-memcached yaml file
	file, err := os.OpenFile("deploy/cr.yaml", os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	_, err = file.WriteString("apiVersion: \"cache.example.com/v1alpha1\"\nkind: \"Memcached\"\nmetadata:\n  name: \"example-memcached\"\nspec:\n  size: 3")
	if err != nil {
		t.Fatal(err)
	}

	file.Close()

	yamlCR, err := ioutil.ReadFile("deploy/cr.yaml")
	memcachedClient := e2eutil.GetCRClient(t, kubeconfig, yamlCR)
	e2eutil.CreateFromYAML(t, yamlCR, kubeclient, kubeconfig, namespace)

	err = e2eutil.DeploymentReplicaCheck(t, kubeclient, namespace, "example-memcached", 3, 6)
	if err != nil {
		t.Fatal(err)
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
		t.Fatal(err)
	}

	err = e2eutil.DeploymentReplicaCheck(t, kubeclient, namespace, "example-memcached", 4, 6)
	if err != nil {
		t.Fatal(err)
	}

	// clean everything up
	err = memcachedClient.Delete().
		Namespace(namespace).
		Resource("memcacheds").
		Name("example-memcached").
		Body([]byte("{\"propagationPolicy\":\"Foreground\"}")).
		Do().
		Error()
	if err != nil {
		t.Log("Failed to delete example-memcached CR")
		t.Fatal(err)
	}
	err = kubeclient.AppsV1().Deployments(namespace).
		Delete("memcached-operator", metav1.NewDeleteOptions(0))
	if err != nil {
		t.Log("Failed to delete memcached-operator deployment")
		t.Fatal(err)
	}
	err = kubeclient.RbacV1beta1().Roles(namespace).Delete("memcached-operator", metav1.NewDeleteOptions(0))
	if err != nil {
		t.Log("Failed to delete memcached-operator Role")
		t.Fatal(err)
	}
	err = kubeclient.RbacV1beta1().RoleBindings(namespace).Delete("default-account-memcached-operator", metav1.NewDeleteOptions(0))
	if err != nil {
		t.Log("Failed to delete memcached-operator RoleBinding")
		t.Fatal(err)
	}
	extensionclient, err := extensions.NewForConfig(kubeconfig)
	if err != nil {
		t.Fatal(err)
	}
	err = extensionclient.ApiextensionsV1beta1().CustomResourceDefinitions().Delete("memcacheds.cache.example.com", metav1.NewDeleteOptions(0))
	if err != nil {
		t.Log("Failed to delete memcached CRD")
		t.Fatal(err)
	}
	err = kubeclient.CoreV1().Namespaces().Delete(namespace, metav1.NewDeleteOptions(0))
	if err != nil {
		t.Log("Failed to delete memcached namespace")
		t.Fatal(err)
	}

	os.RemoveAll(os.Getenv("GOPATH") + "/src/github.com/example-inc/memcached-operator")
}
