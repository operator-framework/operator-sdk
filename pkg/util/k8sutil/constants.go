package k8sutil

const (
	// KubeConfigEnvVar defines the env variable KUBERNETES_CONFIG which
	// contains the kubeconfig file path.
	KubeConfigEnvVar = "KUBERNETES_CONFIG"

	// WatchNamespaceEnvVar is the constant for env variable WATCH_NAMESPACE
	// which is the namespace that the pod is currently running in.
	WatchNamespaceEnvVar = "WATCH_NAMESPACE"

	// OperatorNameEnvVar is the constant for env variable OPERATOR_NAME
	// wich is the name of the current operator
	OperatorNameEnvVar = "OPERATOR_NAME"

	// PrometheusMetricsPort defines the port which expose prometheus metrics
	PrometheusMetricsPort = 60000

	// PrometheusMetricsPortName define the port name used in kubernetes deployment and service
	PrometheusMetricsPortName = "metrics"
)
