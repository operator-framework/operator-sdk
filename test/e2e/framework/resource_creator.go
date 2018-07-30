package framework

import (
	"testing"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (ctx *TestCtx) CreateNamespace(f *Framework, t *testing.T) (string, error) {
	// create namespace
	namespace := ctx.GetObjID()
	namespaceObj := &core.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	_, err := f.KubeClient.CoreV1().Namespaces().Create(namespaceObj)
	if err != nil {
		return "", err
	}
	t.Logf("Created namespace: %s", namespace)
	ctx.AddFinalizerFn(func() error { return f.KubeClient.CoreV1().Namespaces().Delete(namespace, metav1.NewDeleteOptions(0)) })
	return namespace, nil
}
