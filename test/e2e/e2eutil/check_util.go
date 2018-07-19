package e2eutil

import (
	"testing"

	"github.com/operator-framework/operator-sdk/pkg/util/retryutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func DeploymentReplicaCheck(t *testing.T, kubeclient *kubernetes.Clientset, namespace, name string, replicas, retries int) error {
	err := retryutil.Retry(retryInterval, retries, func() (done bool, err error) {
		deployment, err := kubeclient.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{IncludeUninitialized: true})
		if err != nil {
			// sometimes, a deployment has not been created by the time we call this; we
			// assume that is what happened instead of immediately failing
			t.Logf("Waiting for availability of %s deployment\n", name)
			return false, nil
		}

		if int(deployment.Status.AvailableReplicas) == replicas {
			return true, nil
		}
		t.Logf("Waiting for full availability of %s deployment (%d/%d)\n", name, deployment.Status.AvailableReplicas, replicas)
		return false, nil
	})
	if err != nil {
		return err
	}
	t.Logf("Deployment available (%d/%d)\n", replicas, replicas)
	return nil
}
