package framework

import (
	"flag"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"k8s.io/client-go/rest"
)

var Global *Framework

type Framework struct {
	KubeConfig   *rest.Config
	KubeClient   kubernetes.Interface
	ExternalRepo string
}

func setup() error {
	config := flag.String("kubeconfig", os.Getenv("HOME")+"/.kube/config", "kube config path, e.g. $HOME/.kube/config")
	externalRepo := flag.String("external-repo", "", "Repo to push docker image to, e.g. quay.io/example-inc")
	flag.Parse()
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
		ExternalRepo: *externalRepo,
	}
	return nil
}
