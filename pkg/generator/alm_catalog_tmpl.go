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

const catalogPackageTmpl = `packageName: {{.PackageName}}
channels:
- name: {{.ChannelName}}
  currentCSV: {{.CurrentCSV}}
`

const crdTmpl = `apiVersion: apiextensions.k8s.io/v1beta1
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
`

const catalogCSVTmpl = `apiVersion: app.coreos.com/v1alpha1
kind: ClusterServiceVersion-v1
metadata:
  name: {{.CSVName}}
  namespace: placeholder
spec:
  install: 
    strategy: deployment
    spec:
      permissions:
      - serviceAccountName: {{.ProjectName}}
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
      deployments: 
      - name: {{.ProjectName}}
        spec:
          replicas: 1
          selector:
            matchLabels:
              app: {{.ProjectName}}
          template:
            metadata:
              labels:
                app: {{.ProjectName}}
            spec:
              containers:
                - name: {{.ProjectName}}-alm-owned
                  image: {{.Image}}
                  command:
                  - {{.ProjectName}}
                  imagePullPolicy: Always
                  env:
                  - name: MY_POD_NAMESPACE
                    valueFrom:
                      fieldRef:
                        fieldPath: metadata.namespace
                  - name: MY_POD_NAME
                    valueFrom:
                      fieldRef:
                        fieldPath: metadata.name
              restartPolicy: Always
              terminationGracePeriodSeconds: 5
              serviceAccountName: {{.ProjectName}}
              serviceAccount: {{.ProjectName}}
  customresourcedefinitions:
    owned:
      - description: Represents an instance of a {{.Kind}} application
        displayName: {{.Kind}} Application
        kind: {{.Kind}}
        name: {{.KindPlural}}.{{.GroupName}}
        version: {{.CRDVersion}}
  version: {{.CatalogVersion}}
  displayName: {{.Kind}}
  labels:
    alm-owner-enterprise-app: {{.ProjectName}}
    alm-status-descriptors: {{.CSVName}}
`
