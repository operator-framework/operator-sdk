package framework

import (
	"flag"
	"log"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var Global *Framework

type Framework struct {
	KubeConfig *rest.Config
	KubeClient kubernetes.Interface
}

func setup() error {
	homedir, ok := os.LookupEnv("HOME")
	var config *string
	if !ok {
		config = flag.String("kubeconfig", "", "kube config path, e.g. $HOME/.kube/config")
	} else {
		config = flag.String("kubeconfig", homedir+"/.kube/config", "kube config path, e.g. $HOME/.kube/config")
	}
	flag.Parse()
	if *config == "" {
		log.Fatalf("Cannot find kubeconfig, exiting\n")
	}
	kubeconfig, err := clientcmd.BuildConfigFromFlags("", *config)
	if err != nil {
		return err
	}
	kubeclient, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return err
	}
	Global = &Framework{
		KubeConfig: kubeconfig,
		KubeClient: kubeclient,
	}
	return nil
}
