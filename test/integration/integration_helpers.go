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
	"testing"

	apimanifests "github.com/operator-framework/api/pkg/manifests"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const (
	imageEnvVar = "OSDK_INTEGRATION_IMAGE"
)

var (
	// Set with OSDK_INTEGRATION_IMAGE in CI.
	testImageTag = "memcached-operator"
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
	ReplacesCSVName string
	CRDKeys         []DefinitionKey
	InstallModes    []operatorsv1alpha1.InstallMode

	IsBundle bool
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
{{- if .ReplacesCSVName }}
  replaces: {{ .ReplacesCSVName }}
{{- end }}
  version: {{ .OperatorVersion }}
`

func writeOperatorManifests(dir string, csvConfig CSVTemplateConfig) error {
	manifestDir := ""
	if csvConfig.IsBundle {
		manifestDir = filepath.Join(dir, bundle.ManifestsDir)
	} else {
		manifestDir = filepath.Join(dir, csvConfig.OperatorVersion)
	}
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
		crdPath := filepath.Join(manifestDir, fmt.Sprintf("%s.crd.yaml", key.Name))
		if err := writeManifest(crdPath, crd); err != nil {
			return err
		}
	}
	csvPath := ""
	if csvConfig.IsBundle {
		csvPath = filepath.Join(manifestDir, fmt.Sprintf("%s.csv.yaml", csvConfig.OperatorName))
	} else {
		csvPath = filepath.Join(manifestDir, fmt.Sprintf("%s.v%s.csv.yaml",
			csvConfig.OperatorName, csvConfig.OperatorVersion))
	}
	if err := execTemplateOnFile(csvPath, csvTmpl, csvConfig); err != nil {
		return err
	}
	return nil
}

func writePackageManifest(dir, pkgName string, channels []apimanifests.PackageChannel) error {
	pkg := apimanifests.PackageManifest{
		PackageName:        pkgName,
		DefaultChannelName: channels[0].Name,
		Channels:           channels,
	}
	pkgPath := filepath.Join(dir, fmt.Sprintf("%s.package.yaml", pkgName))
	return writeManifest(pkgPath, pkg)
}

func writeManifest(path string, o interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	b, err := yaml.Marshal(o)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, b, 0644)
}

func execTemplateOnFile(path, tmplStr string, o interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	w, err := os.Create(path)
	if err != nil {
		return err
	}
	defer w.Close()
	tmpl, err := template.New(path).Parse(tmplStr)
	if err != nil {
		return err
	}
	return tmpl.Execute(w, o)
}

//nolint:unparam
func mkTempDirWithCleanup(t *testing.T, prefix string) (dir string, f func()) {
	var err error
	if prefix == "" {
		prefix = "sdk-integration-"
	}
	if dir, err = ioutil.TempDir("", prefix); err != nil {
		t.Fatalf("Failed to create tmp dir: %v", err)
	}
	f = func() {
		if err := os.RemoveAll(dir); err != nil {
			// Not a test failure since files in /tmp will eventually get deleted
			t.Logf("Failed to remove tmp dir %s: %v", dir, err)
		}
	}
	return
}
