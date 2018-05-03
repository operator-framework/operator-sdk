package generator

import (
	"bytes"
	"testing"
)

const crdYamlExp = `apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: appservices.app.example.com
spec:
  group: app.example.com
  names:
    kind: AppService
    listKind: AppServiceList
    plural: appservices
    singular: appservice
  scope: Namespaced
  version: v1alpha1
`

const operatorYamlExp = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: app-operator
spec:
  replicas: 1
  selector:
    matchLabels:
      name: app-operator
  template:
    metadata:
      labels:
        name: app-operator
    spec:
      containers:
        - name: app-operator
          image: quay.io/example-inc/app-operator:0.0.1
          command:
          - app-operator
          imagePullPolicy: Always
`

const rbacYamlExp = `kind: Role
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: app-operator
rules:
- apiGroups:
  - app.example.com
  resources:
  - "*"
  verbs:
  - "*"
- apiGroups:
  - ""
  resources:
  - pods
  - services
  - endpoints
  - persistentvolumeclaims
  - events
  - configmaps
  - secrets
  verbs:
  - "*"
- apiGroups:
  - apps
  resources:
  - deployments
  - daemonsets
  - replicasets
  - statefulsets
  verbs:
  - "*"

---

kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: default-account-app-operator
subjects:
- kind: ServiceAccount
  name: default
roleRef:
  kind: Role
  name: app-operator
  apiGroup: rbac.authorization.k8s.io
`

func TestGenDeploy(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := renderCRDYaml(buf, appKind, appAPIVersion); err != nil {
		t.Error(err)
	}
	if crdYamlExp != buf.String() {
		t.Errorf("want %v, got %v", crdYamlExp, buf.String())
	}

	buf = &bytes.Buffer{}
	if err := renderOperatorYaml(buf, appProjectName, appImage); err != nil {
		t.Error(err)
	}
	if operatorYamlExp != buf.String() {
		t.Errorf("want %v, got %v", operatorYamlExp, buf.String())
	}

	buf = &bytes.Buffer{}
	if err := renderRBACYaml(buf, appProjectName, appGroupName); err != nil {
		t.Error(err)
	}
	if rbacYamlExp != buf.String() {
		t.Errorf("want %v, got %v", rbacYamlExp, buf.String())
	}
}
