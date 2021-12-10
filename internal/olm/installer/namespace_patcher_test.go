package installer

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

var converter = runtime.DefaultUnstructuredConverter

func TestNamespacePatcher(t *testing.T) {
	ts := []struct {
		name          string
		object        client.Object
		assertCorrect func(*testing.T, *unstructured.Unstructured)
	}{
		{
			name:   "patch namespaced resource",
			object: newService(DefaultOLMNamespace),
			assertCorrect: func(t *testing.T, actual *unstructured.Unstructured) {
				var service corev1.Service
				if err := converter.FromUnstructured(actual.Object, &service); err != nil {
					t.Fatal(err)
				}

				if service.Namespace != "new-namespace" {
					t.Fatalf("invalid namespace, got %s, want new-namespace", service.Namespace)
				}
			},
		},
		{
			name:   "patch cluster-scoped resource",
			object: newCRD(),
			assertCorrect: func(t *testing.T, actual *unstructured.Unstructured) {
				var crd apiextensions.CustomResourceDefinition
				if err := converter.FromUnstructured(actual.Object, &crd); err != nil {
					t.Fatal(err)
				}

				if crd.Namespace != "" {
					t.Fatalf("invalid namespace, got %s, want empty value", crd.Namespace)
				}
			},
		},
		{
			name:   "patch cluster role binding",
			object: newClusterRoleBinding(),
			assertCorrect: func(t *testing.T, actual *unstructured.Unstructured) {
				var crb rbacv1.ClusterRoleBinding
				if err := converter.FromUnstructured(actual.Object, &crb); err != nil {
					t.Fatal(err)
				}

				if crb.Namespace != "" {
					t.Fatalf("invalid namespace, got %s, want empty value", crb.Namespace)
				}

				if crb.Subjects[0].Namespace != "new-namespace" {
					t.Fatalf("invalid namespace, got %s, want new-namespace", crb.Subjects[0].Namespace)
				}

				if crb.Subjects[1].Namespace != "non-olm-namespace" {
					t.Fatalf("invalid namespace, got %s, want non-olm-namespace", crb.Subjects[1].Namespace)
				}
			},
		},
		{
			name:   "patch deployment for catalog operator",
			object: newDeployment(catalogOperatorName, false),
			assertCorrect: func(t *testing.T, actual *unstructured.Unstructured) {
				var deployment appsv1.Deployment
				if err := converter.FromUnstructured(actual.Object, &deployment); err != nil {
					t.Fatal(err)
				}

				args := deployment.Spec.Template.Spec.Containers[0].Args
				if args[0] != "-namespace" {
					t.Fatalf("invalid arg 0, got %s, want -namespace", args[0])
				}
				if args[1] != "new-namespace" {
					t.Fatalf("invalid arg 1, got %s, want new-namespace", args[1])
				}
			},
		},
		{
			name:   "patch deployment for catalog operator with single line args",
			object: newDeployment(catalogOperatorName, true),
			assertCorrect: func(t *testing.T, actual *unstructured.Unstructured) {
				var deployment appsv1.Deployment
				if err := converter.FromUnstructured(actual.Object, &deployment); err != nil {
					t.Fatal(err)
				}

				args := deployment.Spec.Template.Spec.Containers[0].Args
				if args[0] != "-namespace=new-namespace" {
					t.Fatalf("invalid arg 0, got %s, want -namespace=new-namespace", args[0])
				}
			},
		},
		{
			name:   "patch deployment for regular deployment",
			object: newDeployment("regular deployment", false),
			assertCorrect: func(t *testing.T, actual *unstructured.Unstructured) {
				var deployment appsv1.Deployment
				if err := converter.FromUnstructured(actual.Object, &deployment); err != nil {
					t.Fatal(err)
				}

				args := deployment.Spec.Template.Spec.Containers[0].Args
				if args[0] != "-namespace" {
					t.Fatalf("invalid arg 0, got %s, want -namespace", args[0])
				}
				if args[1] != "olm" {
					t.Fatalf("invalid arg 1, got %s, want olm", args[1])
				}
			},
		},
	}

	for _, tc := range ts {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			u, err := toUnstructured(tc.object)
			if err != nil {
				t.Fatal(err)
			}
			np := newNamespacePatcher()
			actual, err := np.setObjectsNamespace([]unstructured.Unstructured{u}, "new-namespace")
			if err != nil {
				t.Fatal(err)
			}

			if len(actual) != 1 {
				t.Fatalf("invalid response length, got %d, want 1", len(actual))
			}

			tc.assertCorrect(t, &actual[0])
		})
	}
}

func newDeployment(name string, singleLineArgs bool) *appsv1.Deployment {
	var args []string
	if singleLineArgs {
		args = []string{"-namespace=" + DefaultOLMNamespace}
	} else {
		args = []string{"-namespace", DefaultOLMNamespace}
	}
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind: "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "old-namespace",
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Args: args,
						},
					},
				},
			},
		},
	}
}

func newService(namespace string) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind: "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "regular-service",
			Namespace: namespace,
		},
	}
}

func newCRD() *apiextensionsv1.CustomResourceDefinition {
	return &apiextensionsv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind: "CustomResourceDefinition",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "crd",
		},
	}
}

func newClusterRoleBinding() *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind: "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster-role-binding",
		},
		Subjects: []rbacv1.Subject{
			{
				Name:      "subject1",
				Namespace: DefaultOLMNamespace,
			},
			{
				Name:      "subject1",
				Namespace: "non-olm-namespace",
			},
		},
	}
}

func toUnstructured(object client.Object) (unstructured.Unstructured, error) {
	var u unstructured.Unstructured
	o, err := converter.ToUnstructured(object)
	if err != nil {
		return unstructured.Unstructured{}, err
	}
	u.Object = o

	return u, nil
}
