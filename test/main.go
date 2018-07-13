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

	// setup list options
	listOptions := metav1.ListOptions{
		LabelSelector: "",
		FieldSelector: "",
	}

	// create rbac
	output, err := exec.Command("kubectl", "create", "-f", "deploy/rbac.yaml").Output()
	if err != nil {
		fmt.Println("An error occurred")
		fmt.Printf("%s\n", output)
		log.Fatalf("%s\n", err)
	}
	fmt.Println("Created rbac")

	// create crd
	output, err = exec.Command("kubectl", "create", "-f", "deploy/crd.yaml").Output()
	if err != nil {
		fmt.Println("An error occurred")
		fmt.Printf("%s\n", output)
		log.Fatalf("%s\n", err)
	}
	fmt.Println("Created crd")

	// create operator
	output, err = exec.Command("kubectl", "create", "-f", "deploy/operator.yaml").Output()
	if err != nil {
		fmt.Println("An error occurred")
		fmt.Printf("%s\n", output)
		log.Fatalf("%s\n", err)
	}
	fmt.Println("Created operator")

	// get deployments
	deployments, err := api.Deployments("").List(listOptions)
	if err != nil {
		log.Fatal(err)
	}
	// printDeployments(deployments)

	count := 0
oploop:
	for {
		if count >= 60 {
			break
		}
		count++
		deployments, err = api.Deployments("").List(listOptions)
		if err != nil {
			log.Fatal(err)
		}
		for _, deployment := range deployments.Items {
			if deployment.Name == "memcached-operator" && deployment.Status.AvailableReplicas == 1 {
				break oploop
			} else if deployment.Name == "memcached-operator" {
				fmt.Printf("Waiting for full availability of operator deployment (%d/1)\n", deployment.Status.AvailableReplicas)
				// printDeployments(deployments)
				time.Sleep(time.Second * 1)
				continue
			}
		}
	}

	fmt.Println("Deployment available (1/1)")

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
	output, err = exec.Command("kubectl", "apply", "-f", "deploy/cr.yaml").Output()
	if err != nil {
		fmt.Println("An error occurred")
		fmt.Printf("%s\n", output)
		log.Fatalf("%s\n", err)
	}

	count = 0
	// wait for 3 available replicas for example-memcached deployment
sizeloop3:
	for {
		if count >= 60 {
			break
		}
		count++
		deployments, err = api.Deployments("").List(listOptions)
		if err != nil {
			log.Fatal(err)
		}
		for _, deployment := range deployments.Items {
			if deployment.Name == "example-memcached" && deployment.Status.AvailableReplicas == 3 {
				break sizeloop3
			} else if deployment.Name == "example-memcached" {
				fmt.Printf("Waiting for full availability of memcached deployment (%d/3)\n", deployment.Status.AvailableReplicas)
				// printDeployments(deployments)
				time.Sleep(time.Second * 1)
				continue
			}
		}
	}

	fmt.Println("Deployment available (3/3)")
	//	printDeployments(deployments)

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

	_, err = file.WriteString(string(newCr))
	if err != nil {
		log.Fatal(err)
	}

	file.Close()

	// apply updated example-memcached deployment
	output, err = exec.Command("kubectl", "apply", "-f", "deploy/cr.yaml").Output()
	if err != nil {
		fmt.Println("An error occurred")
		fmt.Printf("%s\n", output)
		log.Fatalf("%s\n", err)
	}

	count = 0
	// wait for 4 available replicas for example-memcached deployment
sizeloop4:
	for {
		if count >= 60 {
			break
		}
		count++
		deployments, err = api.Deployments("").List(listOptions)
		if err != nil {
			log.Fatal(err)
		}
		for _, deployment := range deployments.Items {
			if deployment.Name == "example-memcached" && deployment.Status.AvailableReplicas == 4 {
				break sizeloop4
			} else if deployment.Name == "example-memcached" {
				fmt.Printf("Waiting for full availability of memcached deployment (%d/4)\n", deployment.Status.AvailableReplicas)
				// printDeployments(deployments)
				time.Sleep(time.Second * 1)
				continue
			}
		}
	}
	fmt.Println("Deployment available (4/4)")
	// printDeployments(deployments)
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
