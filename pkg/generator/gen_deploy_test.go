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
  labels:
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
          ports:
          - containerPort: 9090
            name: metrics
          command:
          - app-operator
          imagePullPolicy: Always
          env:
            - name: WATCH_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
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

const serviceYamlExp = `apiVersion: v1
kind: Service
metadata:
  name: app-operator
  labels:
    name: app-operator
spec:
  selector:
    name: app-operator
  ports:
  - protocol: TCP
    targetPort: metrics
    port: 9090
    name: metrics`

const serviceMonitorYamlExp = `apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: app-operator
  labels:
    name: app-operator
spec:
  selector:
    matchLabels:
      name: app-operator
  endpoints:
  - port: metrics`

func TestGenDeploy(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := renderCRDYaml(buf, appKind, appAPIVersion); err != nil {
		t.Error(err)
	}
	if crdYamlExp != buf.String() {
		t.Errorf(errorMessage, crdYamlExp, buf.String())
	}

	buf = &bytes.Buffer{}
	if err := renderOperatorYaml(buf, appProjectName, appImage); err != nil {
		t.Error(err)
	}
	if operatorYamlExp != buf.String() {
		t.Errorf(errorMessage, operatorYamlExp, buf.String())
	}

	buf = &bytes.Buffer{}
	if err := renderRBACYaml(buf, appProjectName, appGroupName); err != nil {
		t.Error(err)
	}
	if rbacYamlExp != buf.String() {
		t.Errorf(errorMessage, rbacYamlExp, buf.String())
	}

	buf = &bytes.Buffer{}
	if err := renderServiceYaml(buf, appProjectName); err != nil {
		t.Error(err)
	}
	if serviceYamlExp != buf.String() {
		t.Errorf("want %v, got %v", serviceYamlExp, buf.String())
	}

	buf = &bytes.Buffer{}
	if err := renderServiceMonitorYaml(buf, appProjectName); err != nil {
		t.Error(err)
	}
	if serviceMonitorYamlExp != buf.String() {
		t.Errorf("want %v, got %v", serviceMonitorYamlExp, buf.String())
	}
}
