package client

import (
	"net"
	"os"

	"github.com/coreos/play/pkg/generated/clientset/versioned"
	"k8s.io/client-go/rest"
)

func MustNewInCluster() versioned.Interface {
	cfg, err := InClusterConfig()
	if err != nil {
		panic(err)
	}
	return MustNew(cfg)
}

// MustNew create a new vault client based on the Kubernetes client configuration passed in
func MustNew(cfg *rest.Config) versioned.Interface {
	return versioned.NewForConfigOrDie(cfg)
}

func InClusterConfig() (*rest.Config, error) {
	// Work around https://github.com/kubernetes/kubernetes/issues/40973
	// See https://github.com/coreos/etcd-operator/issues/731#issuecomment-283804819
	if len(os.Getenv("KUBERNETES_SERVICE_HOST")) == 0 {
		addrs, err := net.LookupHost("kubernetes.default.svc")
		if err != nil {
			panic(err)
		}
		os.Setenv("KUBERNETES_SERVICE_HOST", addrs[0])
	}
	if len(os.Getenv("KUBERNETES_SERVICE_PORT")) == 0 {
		os.Setenv("KUBERNETES_SERVICE_PORT", "443")
	}
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
