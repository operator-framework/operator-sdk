package test

import (
	"errors"
	"os"

	extensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var Global *Framework

type Framework struct {
	KubeConfig       *rest.Config
	KubeClient       kubernetes.Interface
	ExtensionsClient *extensions.Clientset
	ImageName        *string
	Namespace        *string
}

func setup() error {
	kubeconfigEnv, ok := os.LookupEnv("TEST_KUBECONFIG")
	if ok != true {
		return errors.New("Missing test environment variable; please run with `operator-sdk` test command")
	}
	kubeconfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigEnv)
	if err != nil {
		return err
	}
	kubeclient, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return err
	}
	extensionsClient, err := extensions.NewForConfig(kubeconfig)
	if err != nil {
		return err
	}
	imageName, ok := os.LookupEnv("TEST_IMAGE")
	if ok != true {
		return errors.New("Missing test environment variable; please run with `operator-sdk` test command")
	}
	namespace, ok := os.LookupEnv("TEST_NAMESPACE")
	if ok != true {
		return errors.New("Missing test environment variable; please run with `operator-sdk` test command")
	}
	Global = &Framework{
		KubeConfig:       kubeconfig,
		KubeClient:       kubeclient,
		ExtensionsClient: extensionsClient,
		ImageName:        &imageName,
		Namespace:        &namespace,
	}
	return nil
}
