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

package generator

const operatorYamlTmpl = `apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: {{.KindPlural}}.{{.GroupName}}
spec:
  group: {{.GroupName}}
  names:
    kind: {{.Kind}}
    listKind: {{.Kind}}List
    plural: {{.KindPlural}}
    singular: {{.KindSingular}}
  scope: Namespaced
  version: {{.Version}}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.ProjectName}}
spec:
  replicas: 1
  selector:
    matchLabels:
      name: {{.ProjectName}}
  template:
    metadata:
      labels:
        name: {{.ProjectName}}
    spec:
      containers:
        - name: {{.ProjectName}}
          image: {{.Image}}
          command:
          - {{.ProjectName}}
          imagePullPolicy: Always
`

const rbacYamlTmpl = `kind: Role
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: {{.ProjectName}}
rules:
- apiGroups:
  - {{.GroupName}}
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
  name: default-account-{{.ProjectName}}
subjects:
- kind: ServiceAccount
  name: default
roleRef:
  kind: Role
  name: {{.ProjectName}}
  apiGroup: rbac.authorization.k8s.io
`

const crYamlTmpl = `apiVersion: "{{.APIVersion}}"
kind: "{{.Kind}}"
metadata:
  name: "example"
`
