// Copyright 2018 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import "testing"

var memcachedNamespaceManExample = `apiVersion: v1
kind: ServiceAccount
metadata:
  name: memcached-operator

---

kind: Role
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: memcached-operator
rules:
- apiGroups:
  - cache.example.com
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
  name: memcached-operator
subjects:
- kind: ServiceAccount
  name: memcached-operator
roleRef:
  kind: Role
  name: memcached-operator
  apiGroup: rbac.authorization.k8s.io

---

apiVersion: apps/v1
kind: Deployment
metadata:
  name: memcached-operator
spec:
  replicas: 1
  selector:
    matchLabels:
      name: memcached-operator
  template:
    metadata:
      labels:
        name: memcached-operator
    spec:
      serviceAccountName: memcached-operator
      containers:
        - name: memcached-operator
          image: quay.io/coreos/operator-sdk-dev:test-framework-operator
          ports:
          - containerPort: 60000
            name: metrics
          command:
          - memcached-operator
          imagePullPolicy: Always
          env:
            - name: WATCH_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            - name: OPERATOR_NAME
              value: "memcached-operator"

`

func TestVerifyDeploymentImage(t *testing.T) {
	if err := verifyDeploymentImage([]byte(memcachedNamespaceManExample), "quay.io/coreos/operator-sdk-dev:test-framework-operator"); err != nil {
		t.Fatalf("verifyDeploymentImage incorrectly reported an error: %v", err)
	}
	if err := verifyDeploymentImage([]byte(memcachedNamespaceManExample), "different-image-name"); err == nil {
		t.Fatal("verifyDeploymentImage did not report an error on an incorrect manifest")
	}
}
