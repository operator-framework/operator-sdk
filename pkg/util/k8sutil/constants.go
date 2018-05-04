package k8sutil

const (
	// KubeConfigEnvVar defines the env variable KUBERNETES_CONFIG which
	// contains the kubeconfig file path.
	KubeConfigEnvVar = "KUBERNETES_CONFIG"

	// WatchNamespaceEnvVar is the constant for env variable WATCH_NAMESPACE
	// which is the namespace that the pod is currently running in.
	WatchNamespaceEnvVar = "WATCH_NAMESPACE"
)
