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

	"k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	kubeconfig := filepath.Join(
		os.Getenv("HOME"), ".kube", "config",
	)
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	// NOTE: Certain features are in different clientsets. For example,
	// Pods would be in CoreV1, not AppsV1
	api := clientset.AppsV1()

	// create rbac
	kubectlWrapper("create", "deploy/rbac.yaml")
	fmt.Println("Created rbac")

	// create crd
	kubectlWrapper("create", "deploy/crd.yaml")
	fmt.Println("Created crd")

	// create operator
	kubectlWrapper("create", "deploy/operator.yaml")
	fmt.Println("Created operator")

	deploymentReplicaCheck(api, "memcached-operator", 1, 60)

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

	// apply/create example-memcached deployment
	kubectlWrapper("apply", "deploy/cr.yaml")

	deploymentReplicaCheck(api, "example-memcached", 3, 60)

	// update deployment to 4 replicas
	cr, err := ioutil.ReadFile("deploy/cr.yaml")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
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

	// apply updated example-memcached deployment
	kubectlWrapper("apply", "deploy/cr.yaml")

	deploymentReplicaCheck(api, "example-memcached", 4, 60)
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

func deploymentReplicaCheck(api appsv1.AppsV1Interface, name string, replicas, timeout int) {
	count := 0

	// setup list options
	listOptions := metav1.ListOptions{
		LabelSelector: "",
		FieldSelector: "",
	}

outerloop:
	for {
		if count >= timeout {
			log.Fatalf("Deployment %s did not produce %d available replicas.\n", name, replicas)
		}
		count++
		deployments, err := api.Deployments("").List(listOptions)
		if err != nil {
			log.Fatal(err)
		}
		for _, deployment := range deployments.Items {
			if deployment.Name == name && int(deployment.Status.AvailableReplicas) == replicas {
				break outerloop
			} else if deployment.Name == name {
				fmt.Printf("Waiting for full availability of memcached deployment (%d/%d)\n", deployment.Status.AvailableReplicas, replicas)
				// printDeployments(deployments)
				time.Sleep(time.Second * 1)
				continue
			}
		}
	}
	fmt.Printf("Deployment available (%d/%d)\n", replicas, replicas)
}

func kubectlWrapper(action, file string) {
	output, err := exec.Command("kubectl", action, "-f", file).Output()
	if err != nil {
		fmt.Println("An error occurred")
		fmt.Printf("%s\n", output)
		log.Fatalf("%s\n", err)
	}
}
