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

package catalog

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/operator-framework/operator-sdk/internal/util/diffutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"

	"github.com/coreos/go-semver/semver"
	"github.com/ghodss/yaml"
	olmApi "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	olmInstall "github.com/operator-framework/operator-lifecycle-manager/pkg/controller/install"
)

const testDataDir = "testdata"

func TestCSV(t *testing.T) {
	buf := &bytes.Buffer{}
	s := &scaffold.Scaffold{
		GetWriter: func(_ string, _ os.FileMode) (io.Writer, error) {
			return buf, nil
		},
	}
	csvVer := "0.1.0"

	err := s.Execute(&input.Config{ProjectName: "app-operator"},
		&CSV{
			CSVVersion:     csvVer,
			DeployDir:      filepath.Join(testDataDir, scaffold.DeployDir),
			ConfigFilePath: filepath.Join(testDataDir, scaffold.OlmCatalogDir, CSVConfigYamlFile),
		},
	)
	if err != nil {
		t.Fatalf("Failed to execute the scaffold: (%v)", err)
	}

	if csvExp != buf.String() {
		diffs := diffutil.Diff(csvExp, buf.String())
		t.Fatalf("Expected vs actual differs.\n%v", diffs)
	}
}

const csvExp = `apiVersion: operators.coreos.com/v1alpha1
kind: ClusterServiceVersion
metadata:
  creationTimestamp: null
  name: app-operator.v0.1.0
  namespace: placeholder
spec:
  apiservicedefinitions: {}
  customresourcedefinitions:
    owned:
    - kind: App
      name: apps.example.com
      version: v1alpha1
    - kind: App
      name: apps.example.com
      version: v1alpha2
  description: Placeholder description
  displayName: App Operator
  install:
    spec:
      deployments:
      - name: app-operator
        spec:
          replicas: 1
          selector:
            matchLabels:
              name: app-operator
          strategy: {}
          template:
            metadata:
              creationTimestamp: null
              labels:
                name: app-operator
            spec:
              containers:
              - command:
                - app-operator
                env:
                - name: WATCH_NAMESPACE
                  valueFrom:
                    fieldRef:
                      fieldPath: metadata.namespace
                - name: OPERATOR_NAME
                  value: app-operator
                image: quay.io/example-org/operator:v0.1.0
                imagePullPolicy: Always
                name: app-operator
                ports:
                - containerPort: 60000
                  name: metrics
                resources: {}
              serviceAccountName: app-operator
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
          - apps
          resources:
          - deployments
          - daemonsets
          - replicasets
          - statefulsets
          verbs:
          - '*'
        - apiGroups:
          - app.example.com
          resources:
          - '*'
          verbs:
          - '*'
        serviceAccountName: app-operator
    strategy: deployment
  maturity: alpha
  provider: {}
  version: 0.1.0
`

func TestUpdateVersion(t *testing.T) {
	csv := new(olmApi.ClusterServiceVersion)
	if err := yaml.Unmarshal([]byte(csvExp), csv); err != nil {
		t.Fatal(err)
	}

	newCSVVer := "0.2.0"
	c := &CSV{
		Input: input.Input{
			ProjectName: "app-operator",
		},
		CSVVersion: newCSVVer,
	}
	if err := c.updateCSVVersions(csv); err != nil {
		t.Fatalf("Update csv with ver %s: (%v)", newCSVVer, err)
	}

	wantedSemver := semver.New(newCSVVer)
	if !csv.Spec.Version.Equal(*wantedSemver) {
		t.Errorf("Wanted csv version %v, got %v", *wantedSemver, csv.Spec.Version)
	}
	wantedName := getCSVName("app-operator", newCSVVer)
	if csv.ObjectMeta.Name != wantedName {
		t.Errorf("Wanted csv name %s, got %s", wantedName, csv.ObjectMeta.Name)
	}

	var resolver *olmInstall.StrategyResolver
	stratInterface, err := resolver.UnmarshalStrategy(csv.Spec.InstallStrategy)
	if err != nil {
		t.Fatal(err)
	}
	strat := stratInterface.(*olmInstall.StrategyDetailsDeployment)
	csvPodImage := strat.DeploymentSpecs[0].Spec.Template.Spec.Containers[0].Image
	// updateCSVVersions should not update podspec image.
	wantedImage := "quay.io/example-org/operator:v0.1.0"
	if csvPodImage != wantedImage {
		t.Errorf("Podspec image changed from %s to %s", wantedImage, csvPodImage)
	}

	wantedReplaces := getCSVName("app-operator", "0.1.0")
	if csv.Spec.Replaces != wantedReplaces {
		t.Errorf("Wanted csv replaces %s, got %s", wantedReplaces, csv.Spec.Replaces)
	}
}

func TestGetDisplayName(t *testing.T) {
	cases := []struct {
		input, wanted string
	}{
		{"Appoperator", "Appoperator"},
		{"appoperator", "Appoperator"},
		{"appoperatoR", "Appoperato R"},
		{"AppOperator", "App Operator"},
		{"appOperator", "App Operator"},
		{"app-operator", "App Operator"},
		{"app-_operator", "App Operator"},
		{"App-operator", "App Operator"},
		{"app-_Operator", "App Operator"},
		{"app--Operator", "App Operator"},
		{"app--_Operator", "App Operator"},
		{"APP", "APP"},
		{"another-AppOperator_againTwiceThrice More", "Another App Operator Again Twice Thrice More"},
	}

	for _, c := range cases {
		dn := getDisplayName(c.input)
		if dn != c.wanted {
			t.Errorf("Wanted %s, got %s", c.wanted, dn)
		}
	}
}
