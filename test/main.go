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

	"github.com/operator-framework/operator-sdk/pkg/util/retryutil"
	"k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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

	// create namespace
	output, err := exec.Command("kubectl", "create", "namespace", namespace).Output()
	if err != nil {
		fmt.Println("An error occurred")
		fmt.Printf("%s\n", output)
		log.Fatalf("%s\n", err)
	}

	// create rbac
	kubectlWrapper("create", namespace, "deploy/rbac.yaml")
	fmt.Println("Created rbac")

	// create crd
	kubectlWrapper("create", namespace, "deploy/crd.yaml")
	fmt.Println("Created crd")

	// create operator
	kubectlWrapper("create", namespace, "deploy/operator.yaml")
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

	kubectlWrapper("apply", namespace, "deploy/cr.yaml")

	err = deploymentReplicaCheck(kubeclient, namespace, "example-memcached", 3, 6)
	if err != nil {
		log.Fatal(err)
	}

	// update CR size to 4
	cr, err := ioutil.ReadFile("deploy/cr.yaml")
	if err != nil {
		log.Fatal(err)
	}

	newCr := bytes.Replace(cr, []byte("size: 3"), []byte("size: 4"), -1)

	file, err = os.OpenFile("deploy/cr.yaml", os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	_, err = file.Write(newCr)
	if err != nil {
		log.Fatal(err)
	}

	file.Close()

	kubectlWrapper("apply", namespace, "deploy/cr.yaml")

	err = deploymentReplicaCheck(kubeclient, namespace, "example-memcached", 4, 6)
	if err != nil {
		log.Fatal(err)
	}

	kubectlWrapper("delete", namespace, "deploy/cr.yaml")
	kubectlWrapper("delete", namespace, "deploy/operator.yaml")
}

func printDeployments(deployments *v1.DeploymentList) {
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
		deployment, err := kubeclient.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return false, err
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
