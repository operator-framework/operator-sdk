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

package e2e

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-registry/pkg/registry"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	imageEnvVar = "OSDK_INTEGRATION_IMAGE"
)

var (
	// Set with OSDK_INTEGRATION_IMAGE in CI.
	defaultTestImageTag = "memcached-operator"
)

type DefinitionKey struct {
	Kind     string
	Name     string
	Group    string
	Versions []apiextv1beta1.CustomResourceDefinitionVersion
}

type CSVTemplateConfig struct {
	OperatorName    string
	OperatorVersion string
	TestImageTag    string
	Maturity        string
	ReplacesCSVName string
	CRDKeys         []DefinitionKey
	InstallModes    []operatorsv1alpha1.InstallMode
}

const csvTmpl = `apiVersion: operators.coreos.com/v1alpha1
kind: ClusterServiceVersion
metadata:
  annotations:
    capabilities: Basic Install
  name: {{ .OperatorName }}.v{{ .OperatorVersion }}
  namespace: placeholder
spec:
  apiservicedefinitions: {}
  customresourcedefinitions:
    owned:
{{- range $i, $crd := .CRDKeys }}{{- range $j, $version := $crd.Versions }}
    - description: Represents a cluster of {{ $crd.Kind }} apps
      displayName: {{ $crd.Kind }} App
      kind: {{ $crd.Kind }}
      name: {{ $crd.Name }}
      resources:
      - kind: Deployment
        version: v1
      - kind: ReplicaSet
        version: v1
      - kind: Pod
        version: v1
      specDescriptors:
      - description: The desired number of member Pods for the deployment.
        displayName: Size
        path: size
        x-descriptors:
        - urn:alm:descriptor:com.tectonic.ui:podCount
      statusDescriptors:
      - description: The current status of the application.
        displayName: Status
        path: phase
        x-descriptors:
        - urn:alm:descriptor:io.kubernetes.phase
      - description: Explanation for the current status of the application.
        displayName: Status Details
        path: reason
        x-descriptors:
        - urn:alm:descriptor:io.kubernetes.phase:reason
      version: {{ $version.Name }}
{{- end }}{{- end }}
  description: Big ol' Operator.
  displayName: {{ .OperatorName }} Application
  install:
    spec:
      deployments:
      - name: {{ .OperatorName }}
        spec:
          replicas: 1
          selector:
            matchLabels:
              name: {{ .OperatorName }}
          strategy: {}
          template:
            metadata:
              labels:
                name: {{ .OperatorName }}
            spec:
              containers:
              - command:
                - {{ .OperatorName }}
                env:
                - name: WATCH_NAMESPACE
                  valueFrom:
                    fieldRef:
                      fieldPath: metadata.annotations['olm.targetNamespaces']
                - name: POD_NAME
                  valueFrom:
                    fieldRef:
                      fieldPath: metadata.name
                - name: OPERATOR_NAME
                  value: {{ .OperatorName }}
                image: {{ .TestImageTag }}
                imagePullPolicy: Never
                name: {{ .OperatorName }}
                resources: {}
              serviceAccountName: {{ .OperatorName }}
      permissions:
      - rules:
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
          - '*'
        - apiGroups:
          - ""
          resources:
          - namespaces
          verbs:
          - get
        - apiGroups:
          - apps
          resources:
          - deployments
          - daemonsets
          - replicasets
          - statefulsets
          verbs:
          - '*'
        - apiGroups:
          - monitoring.coreos.com
          resources:
          - servicemonitors
          verbs:
          - get
          - create
        - apiGroups:
          - apps
          resourceNames:
          - {{ .OperatorName }}
          resources:
          - deployments/finalizers
          verbs:
          - update
        serviceAccountName: {{ .OperatorName }}
    strategy: deployment
  installModes:
{{- range $i, $mode := .InstallModes }}
  - supported: {{ $mode.Supported }}
    type: {{ $mode.Type }}
{{- end }}
  keywords:
  - big
  - ol
  - operator
  maintainers:
  - email: corp@example.com
    name: Some Corp
  maturity: {{ .Maturity }}
  provider:
    name: Example
    url: www.example.com
{{- if .ReplacesCSVName }}
  replaces: {{ .ReplacesCSVName }}
{{- end }}
  version: {{ .OperatorVersion }}
`

func writeOperatorManifests(root, operatorName, defaultChannel string,
	csvConfigs ...CSVTemplateConfig) (manifestsDir string, err error) {
	manifestsDir = filepath.Join(root, operatorName)
	pkg := registry.PackageManifest{
		PackageName:        operatorName,
		DefaultChannelName: defaultChannel,
	}
	for _, csvConfig := range csvConfigs {
		pkg.Channels = append(pkg.Channels, registry.PackageChannel{
			Name:           csvConfig.Maturity,
			CurrentCSVName: fmt.Sprintf("%s.v%s", csvConfig.OperatorName, csvConfig.OperatorVersion),
		})
		bundleDir := filepath.Join(manifestsDir, csvConfig.OperatorVersion)
		for _, key := range csvConfig.CRDKeys {
			crd := apiextv1beta1.CustomResourceDefinition{
				TypeMeta: metav1.TypeMeta{
					APIVersion: apiextv1beta1.SchemeGroupVersion.String(),
					Kind:       "CustomResourceDefinition",
				},
				ObjectMeta: metav1.ObjectMeta{Name: key.Name},
				Spec: apiextv1beta1.CustomResourceDefinitionSpec{
					Names: apiextv1beta1.CustomResourceDefinitionNames{
						Kind:     key.Kind,
						ListKind: key.Kind + "List",
						Singular: strings.ToLower(key.Kind),
						Plural:   strings.ToLower(key.Kind) + "s",
					},
					Group:    key.Group,
					Scope:    "Namespaced",
					Versions: key.Versions,
				},
			}
			crdPath := filepath.Join(bundleDir, fmt.Sprintf("%s.crd.yaml", key.Name))
			if err = writeObjectManifest(crdPath, crd); err != nil {
				return "", err
			}
		}
		csvPath := filepath.Join(bundleDir, fmt.Sprintf("%s.v%s.csv.yaml", csvConfig.OperatorName, csvConfig.OperatorVersion))
		if err = execTemplateOnFile(csvPath, csvTmpl, csvConfig); err != nil {
			return "", err
		}
	}
	pkgPath := filepath.Join(manifestsDir, fmt.Sprintf("%s.package.yaml", operatorName))
	if err = writeObjectManifest(pkgPath, pkg); err != nil {
		return "", err
	}
	return manifestsDir, nil
}

func writeObjectManifest(path string, o interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	b, err := yaml.Marshal(o)
	if err != nil {
		return err
	}
	if err = ioutil.WriteFile(path, b, 0644); err != nil {
		return err
	}
	return nil
}

func execTemplateOnFile(path, tmpl string, o interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	w, err := os.Create(path)
	if err != nil {
		return err
	}
	defer w.Close()
	csvTmpl, err := template.New(path).Parse(tmpl)
	if err != nil {
		return err
	}
	if err = csvTmpl.Execute(w, o); err != nil {
		return err
	}
	return nil
}
