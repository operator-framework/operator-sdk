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
	Version         string
	TestImageTag    string
	ReplacesCSVName string
	CRDKeys         []DefinitionKey
	InstallModes    []operatorsv1alpha1.InstallMode

	IsBundle bool
}

// TODO(estroz): devise a way for "make bundle" to be called, then update the generated bundle with correct
// install modes within integration tests themselves.

const csvTmpl = `apiVersion: operators.coreos.com/v1alpha1
kind: ClusterServiceVersion
metadata:
  annotations:
    capabilities: Basic Install
  name: {{ .OperatorName }}.v{{ .Version }}
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
      clusterPermissions:
      - rules:
        {{- range $i, $crd := .CRDKeys }}
        - apiGroups:
          - {{ $crd.Group }}
          resources:
          - {{ $crd.Kind | tolower }}s
          verbs:
          - create
          - delete
          - get
          - list
          - patch
          - update
          - watch
        - apiGroups:
          - {{ $crd.Group }}
          resources:
          - {{ $crd.Kind | tolower }}s/status
          verbs:
          - get
          - patch
          - update
        {{- end}}
        serviceAccountName: default
      - rules:
        - apiGroups:
          - authentication.k8s.io
          resources:
          - tokenreviews
          verbs:
          - create
        - apiGroups:
          - authorization.k8s.io
          resources:
          - subjectaccessreviews
          verbs:
          - create
        serviceAccountName: default
      deployments:
      - name: {{ .OperatorName }}-controller-manager
        spec:
          replicas: 1
          selector:
            matchLabels:
              control-plane: controller-manager
          strategy: {}
          template:
            metadata:
              labels:
                control-plane: controller-manager
            spec:
              containers:
              - args:
                - --secure-listen-address=0.0.0.0:8443
                - --upstream=http://127.0.0.1:8080/
                - --logtostderr=true
                - --v=10
                image: gcr.io/kubebuilder/kube-rbac-proxy:v0.5.0
                name: kube-rbac-proxy
                ports:
                - containerPort: 8443
                  name: https
                resources: {}
              - args:
                - --metrics-addr=127.0.0.1:8080
                - --enable-leader-election
                command:
                - /manager
                image: {{ .TestImageTag }}
                name: manager
                resources:
                  limits:
                    cpu: 100m
                    memory: 30Mi
                  requests:
                    cpu: 100m
                    memory: 20Mi
              terminationGracePeriodSeconds: 10
      permissions:
      - rules:
        - apiGroups:
          - ""
          resources:
          - configmaps
          verbs:
          - get
          - list
          - watch
          - create
          - update
          - patch
          - delete
        - apiGroups:
          - ""
          resources:
          - events
          verbs:
          - create
          - patch
        serviceAccountName: default
    strategy: deployment
  installModes:
{{- range $i, $mode := .InstallModes }}
  - supported: {{ $mode.Supported }}
    type: {{ $mode.Type }}
{{- end }}
{{- if .ReplacesCSVName }}
  replaces: {{ .ReplacesCSVName }}
{{- end }}
  version: {{ .Version }}
`

func writeOperatorManifests(dir string, csvConfig CSVTemplateConfig) error {
	manifestDir := ""
	if csvConfig.IsBundle {
		manifestDir = filepath.Join(dir, bundle.ManifestsDir)
	} else {
		manifestDir = filepath.Join(dir, csvConfig.Version)
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
		crdPath := filepath.Join(manifestDir, fmt.Sprintf("%s_%ss.yaml", key.Name, strings.ToLower(key.Kind)))
		if err := writeManifest(crdPath, crd); err != nil {
			return err
		}
	}
	csvPath := filepath.Join(manifestDir, fmt.Sprintf("%s.clusterserviceversion.yaml", csvConfig.OperatorName))
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

	tmpl := template.New(path).Funcs(map[string]interface{}{
		"tolower": strings.ToLower,
	})
	if tmpl, err = tmpl.Parse(tmplStr); err != nil {
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
