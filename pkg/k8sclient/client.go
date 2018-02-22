package k8sclient

import (
	"fmt"
	"net"
	"os"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	restMapper *discovery.DeferredDiscoveryRESTMapper
	clientPool dynamic.ClientPool
)

// init initializes the restMapper and clientPool needed to create a resource client dynamically
func init() {
	kubeClient, kubeConfig := mustNewKubeClientAndConfig()
	cachedDiscoveryClient := cached.NewMemCacheClient(kubeClient.Discovery())
	restMapper = discovery.NewDeferredDiscoveryRESTMapper(cachedDiscoveryClient, meta.InterfacesForUnstructured)
	restMapper.Reset()
	kubeConfig.ContentConfig = dynamic.ContentConfig()
	clientPool = dynamic.NewClientPool(kubeConfig, restMapper, dynamic.LegacyAPIPathResolverFunc)
}

// GetResourceClient returns the dynamic client and pluralName for the resource specified by the apiVersion and kind
func GetResourceClient(apiVersion, kind, namespace string) (dynamic.ResourceInterface, string, error) {
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse apiVersion: %v", err)
	}
	gvk := schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    kind,
	}

	client, err := clientPool.ClientForGroupVersionKind(gvk)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get client for GroupVersionKind(%s): %v", gvk.String(), err)
	}
	resource, err := apiResource(gvk, restMapper)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get resource type: %v", err)
	}
	pluralName := resource.Name
	resourceClient := client.Resource(resource, namespace)
	return resourceClient, pluralName, nil
}

// apiResource consults the REST mapper to translate an <apiVersion, kind, namespace> tuple to a metav1.APIResource struct.
func apiResource(gvk schema.GroupVersionKind, restMapper *discovery.DeferredDiscoveryRESTMapper) (*metav1.APIResource, error) {
	mapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to get the resource REST mapping for GroupVersionKind(%s): %v", gvk.String(), err)
	}
	resource := &metav1.APIResource{
		Name:       mapping.Resource,
		Namespaced: mapping.Scope == meta.RESTScopeNamespace,
		Kind:       gvk.Kind,
	}
	return resource, nil
}

// mustNewKubeClientAndConfig returns the in-cluster config and kubernetes client
func mustNewKubeClientAndConfig() (kubernetes.Interface, *rest.Config) {
	cfg, err := inClusterConfig()
	if err != nil {
		panic(err)
	}
	return kubernetes.NewForConfigOrDie(cfg), cfg
}

// inClusterConfig returns the in-cluster config accessible inside a pod
func inClusterConfig() (*rest.Config, error) {
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
	return rest.InClusterConfig()
}
