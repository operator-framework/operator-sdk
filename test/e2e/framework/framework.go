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
	KubeConfig   *rest.Config
	KubeClient   kubernetes.Interface
	ExternalRepo *string
}

func setup() error {
	defaultKubeConfig := ""
	homedir, ok := os.LookupEnv("HOME")
	if ok {
		defaultKubeConfig = homedir + "/.kube/config"
	}
	config := flag.String("kubeconfig", defaultKubeConfig, "kubeconfig path, defaults to $HOME/.kube/config")
	extRepo := flag.String("external_repo", "", "external repo to push to, defaults to none (builds image to local docker repo)")
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
		KubeConfig:   kubeconfig,
		KubeClient:   kubeclient,
		ExternalRepo: extRepo,
	}
	return nil
}
