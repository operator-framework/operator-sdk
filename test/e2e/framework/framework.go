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
	defaultKubeConfig := ""
	homedir, ok := os.LookupEnv("HOME")
	if ok {
		defaultKubeConfig = homedir + "/.kube/config"
	}
	config := flag.String("kubeconfig", defaultKubeConfig, "kubeconfig path, defaults to $HOME/.kube/config")
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
