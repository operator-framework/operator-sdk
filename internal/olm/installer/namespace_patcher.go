package installer

import (
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"strings"
)

type namespacePatcher struct {
	uc runtime.UnstructuredConverter
}

func newNamespacePatcher() *namespacePatcher {
	return &namespacePatcher{
		uc: runtime.DefaultUnstructuredConverter,
	}
}

func (np *namespacePatcher) setObjectsNamespace(objs []unstructured.Unstructured, namespace string) ([]unstructured.Unstructured, error) {
	result := make([]unstructured.Unstructured, 0, len(objs))
	for i := range objs {
		obj := &objs[i]
		if err := np.setNamespace(obj, namespace); err != nil {
			return nil, err
		}

		result = append(result, *obj)
	}

	return result, nil
}

func (np *namespacePatcher) setNamespace(obj *unstructured.Unstructured, namespace string) error {
	objKind := obj.GetObjectKind().GroupVersionKind().Kind
	if obj.GetNamespace() == DefaultOLMNamespace {
		obj.SetNamespace(namespace)
	}

	if objKind == "Namespace" && obj.GetName() == DefaultOLMNamespace {
		obj.SetName(namespace)
	}

	if objKind == "ClusterRoleBinding" {
		err := np.setClusterRoleBindingNamespace(obj, namespace)
		if err != nil {
			return err
		}
	}

	if objKind == "Deployment" && obj.GetName() == catalogOperatorName {
		err := np.setDeploymentArgNamespace(obj, namespace)
		if err != nil {
			return err
		}
	}

	return nil
}

func (np *namespacePatcher) setDeploymentArgNamespace(obj *unstructured.Unstructured, namespace string) error {
	var deploy appsv1.Deployment
	if err := np.uc.FromUnstructured(obj.Object, &deploy); err != nil {
		return err
	}
	args := deploy.Spec.Template.Spec.Containers[0].Args
	for i, arg := range args {
		if arg == "-namespace" || arg == "--namespace" {
			args[i+1] = namespace
			continue
		}

		if strings.HasPrefix(arg, "-namespace=") {
			parts := strings.SplitN(arg, "=", 2)
			args[i] = fmt.Sprintf("%s=%s", parts[0], namespace)
			continue
		}
	}

	o, err := np.uc.ToUnstructured(&deploy)
	if err != nil {
		return err
	}
	obj.Object = o
	return nil
}

func (np *namespacePatcher) setClusterRoleBindingNamespace(obj *unstructured.Unstructured, namespace string) error {
	var crb rbacv1.ClusterRoleBinding
	if err := np.uc.FromUnstructured(obj.Object, &crb); err != nil {
		return err
	}
	for i := range crb.Subjects {
		if crb.Subjects[i].Namespace == DefaultOLMNamespace {
			crb.Subjects[i].Namespace = namespace
		}
	}
	o, err := np.uc.ToUnstructured(&crb)
	if err != nil {
		return err
	}
	obj.Object = o
	return nil
}
